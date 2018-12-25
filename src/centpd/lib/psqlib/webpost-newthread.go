package psqlib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	. "centpd/lib/logx"
	"centpd/lib/mailib"
)

const postTQMsgArgCount = 11
const postTQFileArgCount = 5

func (sp *PSQLIB) getNTStmt(n int) (s *sql.Stmt, err error) {
	sp.ntMutex.RLock()
	s = sp.ntStmts[n]
	sp.ntMutex.RUnlock()

	if s != nil {
		return
	}

	sp.ntMutex.Lock()
	defer sp.ntMutex.Unlock()

	// there couldve been race so re-examine situation
	s = sp.ntStmts[n]
	if s != nil {
		return
	}
	// confirmed no statement is there yet.
	// create new
	var st string
	// header
	sth := `WITH
	ub AS (
		UPDATE
			ib0.boards
		SET
			lastid = lastid + 1,
			t_count = t_count + 1,
			p_count = p_count + 1,
		WHERE
			bid = $1
		RETURNING
			lastid
	),
	ut AS (
		INSERT INTO
			ib0.threads (
				b_id,
				t_id,
				t_name,
				bump,
				p_count,
				f_count
			)
		SELECT
			$1,
			lastid,
			$2,
			$3,
			1,
			$11
		FROM
			ub
	),
	ugp AS (
		INSERT INTO
			ib0.posts (
				pdate,
				padded,
				sage,
				msgid,
				title,
				author,
				trip,
				message,
				headers,
				layout,
				f_count,
			)
		VALUES
			(
				$3,
				NOW(),
				FALSE,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11
			)
		RETURNING
			g_p_id,pdate,padded
	),
	ubp AS (
		INSERT INTO
			ib0.bposts (
				b_id,
				t_id,
				b_p_id,
				p_name,
				g_p_id,
				pdate,
				padded,
				sage
			)
		SELECT
			$1,
			ub.lastid,
			ub.lastid,
			$2,
			ugp.g_p_id,
			ugp.pdate,
			ugp.padded,
			FALSE
		FROM
			ub
		CROSS JOIN
			ugp
	)`
	// footer
	stf := `
SELECT * FROM up`
	if n == 0 {
		st = sth + stf
	} else {
		// dynamically make statement with required places for files
		var b strings.Builder

		st1 := sth + `,
	uf AS (
		INSERT INTO
			ib0.files (
				g_p_id,
				ftype,
				fsize,
				fname,
				thumb,
				oname
			)
		SELECT *
		FROM (
			SELECT g_p_id
			FROM ugp
		) AS q0
		CROSS JOIN (
			VALUES `
		b.WriteString(st1)

		x := postTQMsgArgCount + 1 // counting from 1
		for i := 0; i < n; i++ {
			if i != 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "($%d::ftype_t,$%d::BIGINT,$%d,$%d,$%d)", x+0, x+1, x+2, x+3, x+4)
			x += postTQFileArgCount
		}

		st2 := `
		) AS q1
	)` + stf
		b.WriteString(st2)

		st = b.String()
	}

	sp.log.LogPrintf(DEBUG, "will prepare newthread(%d) statement:\n%s\n", n, st)
	s, err = sp.db.DB.Prepare(st)
	if err != nil {
		return nil, sp.sqlError("newthread statement preparation", err)
	}
	sp.log.LogPrintf(DEBUG, "newthread(%d) statement prepared successfully", n)

	sp.ntStmts[n] = s
	return
}

func (sp *PSQLIB) insertNewThread(
	bid boardID, pInfo mailib.PostInfo) (tid postID, err error) {

	if len(pInfo.H) == 0 {
		panic("post should have header filled")
	}

	stmt, err := sp.getNTStmt(len(pInfo.FI))
	if err != nil {
		return
	}

	Hjson, err := json.Marshal(pInfo.H)
	if err != nil {
		panic(err)
	}
	Ljson, err := json.Marshal(&pInfo.L)
	if err != nil {
		panic(err)
	}

	var r *sql.Row
	if len(pInfo.FI) == 0 {
		r = stmt.QueryRow(
			bid, pInfo.ID, pInfo.Date, pInfo.MessageID,
			pInfo.MI.Title, pInfo.MI.Author,
			pInfo.MI.Trip, pInfo.MI.Message,
			Hjson, Ljson, pInfo.FC)
	} else {
		x := postTQMsgArgCount
		xf := postTQFileArgCount
		args := make([]interface{}, x+(len(pInfo.FI)*xf))
		args[0] = bid
		args[1] = pInfo.ID
		args[2] = pInfo.Date
		args[3] = pInfo.MessageID
		args[4] = pInfo.MI.Title
		args[5] = pInfo.MI.Author
		args[6] = pInfo.MI.Trip
		args[7] = pInfo.MI.Message
		args[8] = Hjson
		args[9] = Ljson
		args[10] = pInfo.FC
		for i := range pInfo.FI {
			args[x+0] = mailib.FTypeS[pInfo.FI[i].Type]
			args[x+1] = pInfo.FI[i].Size
			args[x+2] = pInfo.FI[i].ID
			args[x+3] = pInfo.FI[i].Thumb
			args[x+4] = pInfo.FI[i].Original
			x += xf
		}
		r = stmt.QueryRow(args...)
	}
	err = r.Scan(&tid)
	if err != nil {
		return 0, sp.sqlError("newthread insert query scan", err)
	}

	// done
	return
}
