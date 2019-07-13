package psqlib

import (
	"database/sql"
	"os"

	xtypes "github.com/jmoiron/sqlx/types"

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

func (sp *PSQLIB) preDelete(tx *sql.Tx) (err error) {
	_, err = tx.Exec("LOCK ib0.files IN SHARE ROW EXCLUSIVE MODE")
	if err != nil {
		err = sp.sqlError("lock files query", err)
	}
	return
}

func (sp *PSQLIB) deleteByMsgID(
	tx *sql.Tx, cmsgids CoreMsgIDStr) (err error) {

	err = sp.preDelete(tx)
	if err != nil {
		return
	}

	delst := tx.Stmt(sp.st_prep[st_web_delete_by_msgid])
	rows, err := delst.Query(string(cmsgids))
	if err != nil {
		err = sp.sqlError("delete by msgid query", err)
		return
	}

	err = sp.postDelete(tx, rows)
	return
}

func (sp *PSQLIB) banByMsgID(
	tx *sql.Tx, cmsgids CoreMsgIDStr, banbid boardID, banbpid postID, reason string) (
	err error) {

	err = sp.preDelete(tx)
	if err != nil {
		return
	}

	bidn := sql.NullInt64{
		Int64: int64(banbid),
		Valid: banbid != 0 && banbpid != 0,
	}
	bpidn := sql.NullInt64{
		Int64: int64(banbpid),
		Valid: banbid != 0 && banbpid != 0,
	}

	banst := tx.Stmt(sp.st_prep[st_web_ban_by_msgid])
	rows, err := banst.Query(string(cmsgids), bidn, bpidn, reason)
	if err != nil {
		err = sp.sqlError("ban by msgid query", err)
		return
	}

	err = sp.postDelete(tx, rows)
	return
}

func (sp *PSQLIB) postDelete(tx *sql.Tx, rows *sql.Rows) (err error) {

	type affThr struct {
		b boardID
		t postID
	}
	var thr_aff []affThr

	for rows.Next() {
		var fname, tname string
		var fnum, tnum int64
		var xb_id, xt_id sql.NullInt64
		err = rows.Scan(&fname, &fnum, &tname, &tnum, &xb_id, &xt_id)
		if err != nil {
			rows.Close()
			err = sp.sqlError("delete by msgid rows scan", err)
			return
		}
		// delet
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
		if xb_id.Int64 != 0 && xt_id.Int64 != 0 {
			thr_aff = append(thr_aff, affThr{
				b: boardID(xb_id.Int64),
				t: postID(xt_id.Int64),
			})
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
			toptsst = tx.Stmt(sp.st_prep[st_web_bname_topts_by_tid])
			refbump = tx.Stmt(sp.st_prep[st_web_refresh_bump_by_tid])
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

	return nil
}

func (sp *PSQLIB) DemoDeleteOrBanByMsgID(msgids []string, banreason string) {
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

	for _, s := range msgids {
		sp.log.LogPrintf(INFO, "deleting %s", s)
		if banreason == "" {
			err = sp.deleteByMsgID(tx, cutMsgID(FullMsgIDStr(s)))
		} else {
			err = sp.banByMsgID(tx, cutMsgID(FullMsgIDStr(s)), 0, 0, banreason)
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
