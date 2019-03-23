package psqlib

import (
	"database/sql"
	"os"

	xtypes "github.com/jmoiron/sqlx/types"

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

	type affThr struct {
		b boardID
		t postID
	}
	var thr_aff []affThr

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

	// now de-bump affected threads
	for _, ta := range thr_aff {
		sp.log.LogPrintf(DEBUG, "DEBUMP board %d thread %d", ta.b, ta.t)

		q := `
SELECT
	xb.b_name,xb.thread_opts,xt.thread_opts
FROM
	ib0.boards xb
JOIN
	ib0.threads xt
ON
	xb.b_id = xt.b_id
WHERE
	xb.b_id = $1 AND xt.t_id = $2
`
		var bname string
		var jbTO xtypes.JSONText // board threads options
		var jtTO xtypes.JSONText // thread options
		threadOpts := defaultThreadOptions

		err = tx.
			QueryRow(q, ta.b, ta.t).
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

		q2 := `
UPDATE
	ib0.threads
SET
	bump = pdate
FROM
	(
		SELECT
			pdate
		FROM (
			SELECT
				pdate,
				b_p_id,
				sage
			FROM
				ib0.bposts
			WHERE
				-- count sages against bump limit.
				-- because others do it like that :<
				b_id = $1 AND t_id = $2
			ORDER BY
				pdate ASC,
				b_p_id ASC
			LIMIT
				$3
			-- take bump posts, sorted by original date,
			-- only upto bump limit
		) AS tt
	WHERE
		sage != TRUE
	ORDER BY
		pdate DESC,b_p_id DESC
	LIMIT
		1
	-- and pick latest one
) as xbump
WHERE
	b_id = $1 AND t_id = $2
`
		_, err = tx.Exec(q2, ta.b, ta.t, threadOpts.BumpLimit)
		if err != nil {
			err = sp.sqlError("thread debump exec", err)
			return
		}
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
