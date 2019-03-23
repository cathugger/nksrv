package psqlib

import (
	"database/sql"
	"os"

	. "centpd/lib/logx"
	mm "centpd/lib/minimail"
)

func (sp *PSQLIB) deleteByMsgID(tx *sql.Tx, cmsgids CoreMsgIDStr) (err error) {
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

	tx.Exec("LOCK ib0.files IN SHARE ROW EXCLUSIVE MODE")
	delst := tx.Stmt(sp.st_prep[st_Web_delete_by_msgid])
	rows, err := delst.Query(string(cmsgids))
	if err != nil {
		err = sp.sqlError("delete by msgid query", err)
		return
	}
	for rows.Next() {
		var fname, tname string
		var fnum, tnum int64
		err = rows.Scan(&fname, &fnum, &tname, &tnum)
		if err != nil {
			rows.Close()
			err = sp.sqlError("delete by msgid rows scan", err)
			return
		}
		// delet
		if fnum == 0 {
			err = os.Remove(sp.src.Main() + fname)
			if err != nil && os.IsNotExist(err) {
				err = nil
			}
			if err != nil {
				rows.Close()
				return
			}
		}
		if tnum == 0 {
			err = os.Remove(sp.thm.Main() + tname)
			if err != nil && os.IsNotExist(err) {
				err = nil
			}
			if err != nil {
				rows.Close()
				return
			}
		}
	}
	if err = rows.Err(); err != nil {
		err = sp.sqlError("delete by msgid rows err", err)
		return
	}

	return nil
}

func (sp *PSQLIB) DemoDeleteByMsgID(msgids []string) {
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
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, s := range msgids {
		sp.log.LogPrintf(INFO, "deleting %s", s)
		err = sp.deleteByMsgID(tx, cutMsgID(FullMsgIDStr(s)))
		if err != nil {
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = sp.sqlError("tx commit", err)
		return
	}
}
