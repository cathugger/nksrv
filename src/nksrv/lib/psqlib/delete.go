package psqlib

import (
	"database/sql"
	"os"

	"nksrv/lib/cacheengine"
	. "nksrv/lib/logx"
	mm "nksrv/lib/minimail"
)

// NOTE: nuking OP also nukes whole thread
// OLD {{{
// NOTE:
// because of file dedup we do,
// we must write-lock whole table to avoid new posts adding new entries
// and thus successfuly completing their file move,
// because we gonna delete files we extracted
// TODO:
// above mentioned method sorta sucks for concurrency.
// maybe we should instead do dedup inside DB and count references and
// that would then provide row-level locks?
// idk if and how really that'd work.
// or we could just not bother with it and leave it for filesystem.
// }}}
//
// we don't need locking because we do dedup counting inside database now

type delMsgHandle struct {
	id string
	h  *cacheengine.CacheObj
}

type delMsgIDState struct {
	delmsgids []delMsgHandle
}

func (s delMsgIDState) isNotPresent(id string) bool {
	for i := range s.delmsgids {
		if s.delmsgids[i].id == id {
			return false
		}
	}
	return true
}

type delModIDState struct {
	delmodids map[uint64]struct{}
}

func (s *delModIDState) add(x uint64) {
	if s.delmodids == nil {
		s.delmodids = make(map[uint64]struct{})
	}
	s.delmodids[x] = struct{}{}
}

func (s delModIDState) contain(x uint64) bool {
	_, doesit := s.delmodids[x]
	return doesit
}

func (sp *PSQLIB) deleteByMsgID(
	tx *sql.Tx, cmsgids TCoreMsgIDStr,
	in_delmsgids delMsgIDState, in_delmodids delModIDState) (
	out_delmsgids delMsgIDState, out_delmodids delModIDState,
	err error) {

	sp.log.LogPrintf(DEBUG, "DELET ARTICLE <%s> start", cmsgids)
	delst := tx.Stmt(sp.StPrep[pibase.St_mod_delete_by_msgid])
	_, err = delst.Exec(string(cmsgids))
	if err != nil {
		err = sp.SQLError("delete by msgid query", err)
		return
	}

	sp.log.LogPrintf(DEBUG, "DELET ARTICLE <%s> processing", cmsgids)
	out_delmsgids, out_delmodids, err =
		sp.postDelete(tx, in_delmsgids, in_delmodids)
	sp.log.LogPrintf(DEBUG, "DELET ARTICLE <%s> end", cmsgids)
	return
}

func (sp *PSQLIB) banByMsgID(
	tx *sql.Tx, cmsgids TCoreMsgIDStr,
	banbid boardID, banbpid postID, reason string,
	in_delmsgids delMsgIDState, in_delmodids delModIDState) (
	out_delmsgids delMsgIDState, out_delmodids delModIDState,
	err error) {

	bidn := sql.NullInt64{
		Int64: int64(banbid),
		Valid: banbid != 0 && banbpid != 0,
	}
	bpidn := sql.NullInt64{
		Int64: int64(banbpid),
		Valid: banbid != 0 && banbpid != 0,
	}

	sp.log.LogPrintf(
		DEBUG, "BAN ARTICLE <%s> (reason: %q) start", cmsgids, reason)
	banst := tx.Stmt(sp.StPrep[pibase.St_mod_ban_by_msgid])
	_, err = banst.Exec(string(cmsgids), bidn, bpidn, reason)
	if err != nil {
		err = sp.SQLError("ban by msgid query", err)
		return
	}

	sp.log.LogPrintf(DEBUG, "BAN ARTICLE <%s> processing", cmsgids)
	out_delmsgids, out_delmodids, err =
		sp.postDelete(tx, in_delmsgids, in_delmodids)
	sp.log.LogPrintf(DEBUG, "BAN ARTICLE <%s> end", cmsgids)
	return
}

func (sp *PSQLIB) removeSrcFile(fname string) (err error) {
	err = os.Remove(sp.src.Main() + fname)
	if err != nil && os.IsNotExist(err) {
		err = nil
	}
	return
}

func (sp *PSQLIB) removeThmFile(fname, tname string) (err error) {
	err = os.Remove(sp.thm.Main() + fname + "." + tname)
	if err != nil && os.IsNotExist(err) {
		err = nil
	}
	return
}

func (sp *PSQLIB) postDelete(
	tx *sql.Tx,
	_in_delmsgids delMsgIDState, _in_delmodids delModIDState) (
	out_delmsgids delMsgIDState, out_delmodids delModIDState,
	err error) {

	out_delmsgids = _in_delmsgids
	out_delmodids = _in_delmodids

	var rows *sql.Rows

	// global msgids for cached netnews messages invalidation
	rows, err = sp.drainDelGPosts(tx)
	if err != nil {
		return
	}
	for rows.Next() {
		var msgid string

		err = rows.Scan(&msgid)
		if err != nil {
			rows.Close()
			err = sp.SQLError("rows.Scan", err)
			return
		}

		if out_delmsgids.isNotPresent(msgid) {
			sp.log.LogPrintf(DEBUG, "DELET cached NNTP <%s>", msgid)

			var h *cacheengine.CacheObj
			h, err = sp.nntpce.RemoveItemStart(msgid)
			if err != nil {
				// XXX wrap error?
				rows.Close()
				return
			}
			// XXX can grow large (DoS vector?)
			out_delmsgids.delmsgids = append(out_delmsgids.delmsgids,
				delMsgHandle{
					id: msgid,
					h:  h,
				})
		} else {
			// XXX this shouldn't happen
			sp.log.LogPrintf(
				DEBUG,
				"DELET cached NNTP <%s> (ignored duplicate)", msgid)
		}
	}
	if err = rows.Err(); err != nil {
		err = sp.SQLError("drainDelMsgIDs rows", err)
		return
	}

	// board post infos for references invalidation
	// XXX maybe we could just queue job for this inside trigger?

	type affBPost struct {
		bn string
		pn string
		mi string
	}
	var bp_affs []affBPost

	rows, err = sp.drainDelGPosts(tx)
	if err != nil {
		return
	}
	for rows.Next() {
		var aff affBPost

		err = rows.Scan(&aff.bn, &aff.pn, &aff.mi)
		if err != nil {
			rows.Close()
			err = sp.SQLError("rows.Scan", err)
			return
		}

		bp_affs = append(bp_affs, aff)
	}
	if err = rows.Err(); err != nil {
		err = sp.SQLError("drainDelGPosts rows", err)
		return
	}

	// file deletion
	rows, err = sp.drainDelFiles(tx)
	if err != nil {
		return
	}
	for rows.Next() {
		var fname string

		err = rows.Scan(&fname)
		if err != nil {
			rows.Close()
			err = sp.SQLError("rows.Scan", err)
			return
		}

		sp.log.LogPrintf(DEBUG, "DELET file %q", fname)
		err = sp.removeSrcFile(fname)
		if err != nil {
			rows.Close()
			return
		}
	}

	// thumbnail deletion
	rows, err = sp.drainDelFThumbs(tx)
	if err != nil {
		return
	}
	for rows.Next() {
		var fname, tname string

		err = rows.Scan(&fname, &tname)
		if err != nil {
			rows.Close()
			err = sp.SQLError("rows.Scan", err)
			return
		}

		sp.log.LogPrintf(DEBUG, "DELET thumb %q %q", fname, tname)
		err = sp.removeThmFile(fname, tname)
		if err != nil {
			rows.Close()
			return
		}
	}

	// modids of nuked bposts
	rows, err = sp.drainDelModIDs(tx)
	if err != nil {
		return
	}
	for rows.Next() {
		var mod_id uint64

		err = rows.Scan(&mod_id)
		if err != nil {
			rows.Close()
			err = sp.SQLError("rows.Scan", err)
			return
		}

		out_delmodids.add(mod_id)
	}
	if err = rows.Err(); err != nil {
		err = sp.SQLError("drainDelGPosts rows", err)
		return
	}

	// re-calculate affected references
	xref_up_st := tx.Stmt(sp.StPrep[pibase.St_mod_update_bpost_activ_refs])
	for _, bpa := range bp_affs {
		err = sp.fixupAffectedXRefsInTx(tx, bpa.pn, bpa.bn, TCoreMsgIDStr(bpa.mi), xref_up_st)
		if err != nil {
			return
		}
	}

	return
}

func (sp *PSQLIB) cleanDeletedMsgIDs(delmsgids delMsgIDState) {
	sp.log.LogPrintf(DEBUG, "CLR DEL MSGIDS start")
	for _, x := range delmsgids.delmsgids {
		sp.log.LogPrintf(DEBUG, "CLR DEL MSGIDS <%s>", x.id)
		sp.nntpce.RemoveItemFinish(x.id, x.h)
	}
	sp.log.LogPrintf(DEBUG, "CLR DEL MSGIDS done")
}

func (sp *PSQLIB) DemoDeleteOrBanByMsgID(
	msgids []string, banreason string) {

	var err error

	for _, s := range msgids {
		if !mm.ValidMessageIDStr(TFullMsgIDStr(s)) {
			sp.log.LogPrintf(ERROR, "invalid msgid %q", s)
			return
		}
	}

	tx, err := sp.db.DB.Begin()
	if err != nil {
		err = sp.SQLError("tx begin", err)
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	err = sp.makeDelTables(tx)
	if err != nil {
		return
	}

	var delmsgids delMsgIDState
	defer func() { sp.cleanDeletedMsgIDs(delmsgids) }()

	for _, s := range msgids {
		sp.log.LogPrintf(INFO, "deleting %s", s)
		if banreason == "" {
			delmsgids, _, err =
				sp.deleteByMsgID(tx, cutMsgID(TFullMsgIDStr(s)),
					delmsgids, delModIDState{})
		} else {
			delmsgids, _, err =
				sp.banByMsgID(
					tx, cutMsgID(TFullMsgIDStr(s)), 0, 0, banreason,
					delmsgids, delModIDState{})
		}
		if err != nil {
			sp.log.LogPrintf(ERROR, "%v", err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = sp.SQLError("tx commit", err)
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}
}
