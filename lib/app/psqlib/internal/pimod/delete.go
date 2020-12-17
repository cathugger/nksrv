package pimod

import (
	"database/sql"

	. "nksrv/lib/utils/logx"
	mm "nksrv/lib/utils/minimail"

	. "nksrv/lib/app/psqlib/internal/pibase"
	. "nksrv/lib/app/psqlib/internal/pibasenntp"
	"nksrv/lib/app/psqlib/internal/pirefs"
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

func (mc *modCtx) deleteByMsgID(cmsgids TCoreMsgIDStr) (err error) {

	mc.sp.Log.LogPrintf(DEBUG, "DELET ARTICLE <%s> start", cmsgids)
	delst := mc.tx.Stmt(mc.sp.StPrep[St_mod_delete_by_msgid])
	_, err = delst.Exec(string(cmsgids))
	if err != nil {
		err = mc.sp.SQLError("delete by msgid query", err)
		return
	}

	mc.sp.Log.LogPrintf(DEBUG, "DELET ARTICLE <%s> processing", cmsgids)
	err = mc.postDelete()
	mc.sp.Log.LogPrintf(DEBUG, "DELET ARTICLE <%s> end", cmsgids)
	return
}

func (mc *modCtx) BanByMsgID(
	cmsgids TCoreMsgIDStr,
	banbid TBoardID, banbpid TPostID, reason string,
) (
	err error,
) {

	bidn := sql.NullInt64{
		Int64: int64(banbid),
		Valid: banbid != 0 && banbpid != 0,
	}
	bpidn := sql.NullInt64{
		Int64: int64(banbpid),
		Valid: banbid != 0 && banbpid != 0,
	}

	mc.sp.Log.LogPrintf(
		DEBUG, "BAN ARTICLE <%s> (reason: %q) start", cmsgids, reason)
	banst := mc.tx.Stmt(mc.sp.StPrep[St_mod_ban_by_msgid])
	_, err = banst.Exec(string(cmsgids), bidn, bpidn, reason)
	if err != nil {
		err = mc.sp.SQLError("ban by msgid query", err)
		return
	}

	mc.sp.Log.LogPrintf(DEBUG, "BAN ARTICLE <%s> processing", cmsgids)
	err = mc.postDelete()
	mc.sp.Log.LogPrintf(DEBUG, "BAN ARTICLE <%s> end", cmsgids)
	return
}

func (mc *modCtx) postDelete() (err error) {

	var rows *sql.Rows

	/*
		// global msgids for cached netnews messages invalidation
		rows, err = DrainDelGPosts(sp, tx)
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
				sp.Log.LogPrintf(DEBUG, "DELET cached NNTP <%s>", msgid)

				var h *cacheengine.CacheObj
				h, err = sp.NNTPCE.RemoveItemStart(msgid)
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
				sp.Log.LogPrintf(
					DEBUG,
					"DELET cached NNTP <%s> (ignored duplicate)", msgid)
			}
		}
		if err = rows.Err(); err != nil {
			err = sp.SQLError("drainDelMsgIDs rows", err)
			return
		}
	*/

	// board post infos for references invalidation
	// XXX maybe we could just queue job for this inside trigger?

	type affBPost struct {
		bn string
		pn string
		mi string
	}
	var bp_affs []affBPost

	rows, err = mc.drainDelBPosts()
	if err != nil {
		return
	}
	for rows.Next() {
		var aff affBPost

		err = rows.Scan(&aff.bn, &aff.pn, &aff.mi)
		if err != nil {
			rows.Close()
			err = mc.sp.SQLError("rows.Scan", err)
			return
		}

		bp_affs = append(bp_affs, aff)
	}
	if err = rows.Err(); err != nil {
		err = mc.sp.SQLError("drainDelGPosts rows", err)
		return
	}

	/*
		// modids of nuked bposts
		rows, err = DrainDelModIDs(sp, tx)
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
	*/

	// re-calculate affected references
	xref_up_st := mc.tx.Stmt(mc.sp.StPrep[St_mod_update_bpost_activ_refs])
	for _, bpa := range bp_affs {
		err = pirefs.FixupAffectedXRefsInTx(
			mc.sp, mc.tx, bpa.pn, bpa.bn, TCoreMsgIDStr(bpa.mi), xref_up_st)
		if err != nil {
			return
		}
	}

	return
}

func CleanDeletedMsgIDs(sp *PSQLIB, delmsgids delMsgIDState) {
	sp.Log.LogPrintf(DEBUG, "CLR DEL MSGIDS start")
	for _, x := range delmsgids.delmsgids {
		sp.Log.LogPrintf(DEBUG, "CLR DEL MSGIDS <%s>", x.id)
		sp.NNTPCE.RemoveItemFinish(x.id, x.h)
	}
	sp.Log.LogPrintf(DEBUG, "CLR DEL MSGIDS done")
}

func DemoDeleteOrBanByMsgID(
	sp *PSQLIB,
	msgids []string, banreason string) {

	var err error

	for _, s := range msgids {
		if !mm.ValidMessageIDStr(TFullMsgIDStr(s)) {
			sp.Log.LogPrintf(ERROR, "invalid msgid %q", s)
			return
		}
	}

	tx, err := sp.DB.DB.Begin()
	if err != nil {
		err = sp.SQLError("tx begin", err)
		sp.Log.LogPrintf(ERROR, "%v", err)
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	err = MakeDelTables(sp, tx)
	if err != nil {
		return
	}

	var delmsgids delMsgIDState
	defer func() { CleanDeletedMsgIDs(sp, delmsgids) }()

	for _, s := range msgids {
		sp.Log.LogPrintf(INFO, "deleting %s", s)
		if banreason == "" {
			delmsgids, _, err =
				DeleteByMsgID(
					sp, tx, cutMsgID(TFullMsgIDStr(s)),
					delmsgids, delModIDState{})
		} else {
			delmsgids, _, err =
				BanByMsgID(
					sp,
					tx, cutMsgID(TFullMsgIDStr(s)), 0, 0, banreason,
					delmsgids, delModIDState{})
		}
		if err != nil {
			sp.Log.LogPrintf(ERROR, "%v", err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = sp.SQLError("tx commit", err)
		sp.Log.LogPrintf(ERROR, "%v", err)
		return
	}
}
