package psqlib

import (
	"database/sql"
	"os"

	xtypes "github.com/jmoiron/sqlx/types"

	"nksrv/lib/cacheengine"
	. "nksrv/lib/logx"
	mm "nksrv/lib/minimail"
)

// NOTE: nuking OP also nukes whole thread
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

func (sp *PSQLIB) preModLockFiles(tx *sql.Tx) (err error) {

	sp.log.LogPrintf(DEBUG, "pre-mod LOCK of ib0.files")

	_, err = tx.Exec("LOCK ib0.files IN SHARE ROW EXCLUSIVE MODE")
	if err != nil {
		err = sp.sqlError("lock files query", err)
	}
	return
}

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
	tx *sql.Tx, cmsgids CoreMsgIDStr,
	in_delmsgids delMsgIDState, in_delmodids delModIDState) (
	out_delmsgids delMsgIDState, out_delmodids delModIDState,
	err error) {

	sp.log.LogPrintf(DEBUG, "DELET ARTICLE <%s> start", cmsgids)
	delst := tx.Stmt(sp.st_prep[st_mod_delete_by_msgid])
	rows, err := delst.Query(string(cmsgids))
	if err != nil {
		err = sp.sqlError("delete by msgid query", err)
		return
	}

	sp.log.LogPrintf(DEBUG, "DELET ARTICLE <%s> processing", cmsgids)
	out_delmsgids, out_delmodids, err =
		sp.postDelete(tx, rows, in_delmsgids, in_delmodids)
	sp.log.LogPrintf(DEBUG, "DELET ARTICLE <%s> end", cmsgids)
	return
}

func (sp *PSQLIB) banByMsgID(
	tx *sql.Tx, cmsgids CoreMsgIDStr,
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
	banst := tx.Stmt(sp.st_prep[st_mod_ban_by_msgid])
	rows, err := banst.Query(string(cmsgids), bidn, bpidn, reason)
	if err != nil {
		err = sp.sqlError("ban by msgid query", err)
		return
	}

	sp.log.LogPrintf(DEBUG, "BAN ARTICLE <%s> processing", cmsgids)
	out_delmsgids, out_delmodids, err =
		sp.postDelete(tx, rows, in_delmsgids, in_delmodids)
	sp.log.LogPrintf(DEBUG, "BAN ARTICLE <%s> end", cmsgids)
	return
}

func (sp *PSQLIB) postDelete(
	tx *sql.Tx, rows *sql.Rows,
	_in_delmsgids delMsgIDState, _in_delmodids delModIDState) (
	out_delmsgids delMsgIDState, out_delmodids delModIDState,
	err error) {

	out_delmsgids = _in_delmsgids
	out_delmodids = _in_delmodids

	type affThr struct {
		b   boardID
		t   postID
		loc int64
	}
	var thr_aff []affThr

	type affBPosts struct {
		bn string
		pn string
		mi CoreMsgIDStr
	}
	var bp_aff []affBPosts

	for rows.Next() {
		var fname, tname string
		var fnum, tnum int64
		var xb_id, xt_id sql.NullInt64
		var xt_loc sql.NullInt64
		var msgid sql.NullString
		var b_name sql.NullString
		var p_name sql.NullString
		var p_msgid sql.NullString
		var mod_id sql.NullInt64

		err = rows.Scan(
			&fname, &fnum, &tname, &tnum,
			&xb_id, &xt_id, &xt_loc,
			&msgid, &b_name, &p_name, &p_msgid, &mod_id)
		if err != nil {
			rows.Close()
			err = sp.sqlError("delete by msgid rows scan", err)
			return
		}
		// file and thumb names
		if fname != "" {
			sp.log.LogPrintf(DEBUG, "MAYB DELET file %q num %d", fname, fnum)
		}
		if fname != "" && fnum == 0 {
			sp.log.LogPrintf(DEBUG, "DELET file %q", fname)
			err = os.Remove(sp.src.Main() + fname)
			if err != nil && os.IsNotExist(err) {
				err = nil
			}
			if err != nil {
				rows.Close()
				return
			}
		}
		if tname != "" {
			sp.log.LogPrintf(DEBUG, "MAYB DELET thumb %q num %d", tname, tnum)
		}
		if tname != "" && tnum == 0 {
			sp.log.LogPrintf(DEBUG, "DELET thumb %q", tname)
			err = os.Remove(sp.thm.Main() + tname)
			if err != nil && os.IsNotExist(err) {
				err = nil
			}
			if err != nil {
				rows.Close()
				return
			}
		}
		// affected thread(s) which will need bump recalculated
		if xb_id.Int64 != 0 && xt_id.Int64 != 0 {
			// this won't grow large as crosspost aint so allowing
			thr_aff = append(thr_aff, affThr{
				b:   boardID(xb_id.Int64),
				t:   postID(xt_id.Int64),
				loc: xt_loc.Int64,
			})
		}
		// message-ids which were delet'd, we need to delet them from cache too
		if msgid.String != "" {

			if out_delmsgids.isNotPresent(msgid.String) {

				sp.log.LogPrintf(DEBUG, "DELET cached NNTP <%s>", msgid.String)

				var h *cacheengine.CacheObj
				h, err = sp.nntpce.RemoveItemStart(msgid.String)
				if err != nil {
					// XXX wrap error?
					rows.Close()
					return
				}
				// XXX can grow large (DoS vector?)
				out_delmsgids.delmsgids = append(out_delmsgids.delmsgids,
					delMsgHandle{
						id: msgid.String,
						h:  h,
					})
			} else {
				sp.log.LogPrintf(
					DEBUG,
					"DELET cached NNTP <%s> (ignored duplicate)", msgid.String)
			}
		}
		// deleted publicboardposts
		if b_name.String != "" && p_name.String != "" {
			bp_aff = append(bp_aff, affBPosts{
				bn: b_name.String,
				pn: p_name.String,
				mi: CoreMsgIDStr(p_msgid.String),
			})
		}
		if mod_id.Int64 != 0 {
			sp.log.LogPrintf(DEBUG, "DETEL add modid %d", mod_id.Int64)
			out_delmodids.add(uint64(mod_id.Int64))
		}
	}
	if err = rows.Err(); err != nil {
		err = sp.sqlError("delete by msgid rows err", err)
		return
	}

	// thread opts, refresh bump statements
	var toptsst, refbump *sql.Stmt

	// now de-bump affected threads
	for _, ta := range thr_aff {
		sp.log.LogPrintf(DEBUG, "DEBUMP board %d thread %d", ta.b, ta.t)

		if toptsst == nil {
			toptsst = tx.Stmt(sp.st_prep[st_mod_bname_topts_by_tid])
			refbump = tx.Stmt(sp.st_prep[st_mod_refresh_bump_by_tid])
		}

		var bname string
		var jbTO xtypes.JSONText // board threads options
		var jtTO xtypes.JSONText // thread options
		threadOpts := defaultThreadOptions

		// first obtain thread opts to figure out bump limit
		err = toptsst.
			QueryRow(ta.b, ta.t).
			Scan(&bname, &jbTO, &jtTO)
		if err != nil {
			if err == sql.ErrNoRows {
				sp.log.LogPrintf(DEBUG, "DEBUMP boardthread missing wtf")
				// just skip it
				continue
			}
			err = sp.sqlError("board x thread row query scan", err)
			return
		}

		err = sp.unmarshalBoardThreadOpts(&threadOpts, jbTO, jtTO)
		if err != nil {
			return
		}
		sp.applyInstanceThreadOptions(&threadOpts, bname)

		// perform bump refresh
		_, err = refbump.Exec(ta.b, ta.t, threadOpts.BumpLimit)
		if err != nil {
			err = sp.sqlError("thread debump exec", err)
			return
		}
	}

	// re-calculate affected references
	xref_up_st := tx.Stmt(sp.st_prep[st_mod_update_bpost_activ_refs])
	for _, bpa := range bp_aff {
		err = sp.fixupAffectedXRefsInTx(tx, bpa.pn, bpa.bn, bpa.mi, xref_up_st)
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
		if !mm.ValidMessageIDStr(FullMsgIDStr(s)) {
			sp.log.LogPrintf(ERROR, "invalid msgid %q", s)
			return
		}
	}

	tx, err := sp.db.DB.Begin()
	if err != nil {
		err = sp.sqlError("tx begin", err)
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	err = sp.preModLockFiles(tx)
	if err != nil {
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}

	var delmsgids delMsgIDState
	defer func() { sp.cleanDeletedMsgIDs(delmsgids) }()

	for _, s := range msgids {
		sp.log.LogPrintf(INFO, "deleting %s", s)
		if banreason == "" {
			delmsgids, _, err =
				sp.deleteByMsgID(tx, cutMsgID(FullMsgIDStr(s)),
					delmsgids, delModIDState{})
		} else {
			delmsgids, _, err =
				sp.banByMsgID(
					tx, cutMsgID(FullMsgIDStr(s)), 0, 0, banreason,
					delmsgids, delModIDState{})
		}
		if err != nil {
			sp.log.LogPrintf(ERROR, "%v", err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = sp.sqlError("tx commit", err)
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}
}
