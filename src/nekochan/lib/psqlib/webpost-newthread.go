package psqlib

import (
	"database/sql"
	"fmt"
	"strings"

	. "nekochan/lib/logx"
)

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
		UPDATE ib0.boards
		SET lastid = lastid+1
		WHERE bid = $1
		RETURNING lastid
	),
	ut AS (
		INSERT INTO ib0.threads (bid,tid,tname,bump)
		SELECT $1,lastid,$2,$3
		FROM ub
	),
	up AS (
		INSERT INTO ib0.posts (bid,tid,pid,pname,pdate,padded,sage,msgid,title,author,trip,message)
		SELECT $1,lastid,lastid,$2,$3,NOW(),FALSE,$4,$5,$6,$7,$8
		FROM ub
		RETURNING pid
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
		INSERT INTO ib0.files (bid,pid,fname,thumb,oname)
		SELECT *
		FROM (
			SELECT $1,pid
			FROM up
		) AS q0
		CROSS JOIN (
			VALUES `
		b.WriteString(st1)

		x := 9 // 8 args already, counting from 1
		for i := 0; i < n; i++ {
			if i != 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "($%d, $%d, $%d)", x+0, x+1, x+2)
			x += 3
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

func (sp *PSQLIB) insertNewThread(bid boardID, pInfo postInfo,
	fileInfos []fileInfo) (tid postID, err error) {

	stmt, err := sp.getNTStmt(len(fileInfos))
	if err != nil {
		return
	}

	var r *sql.Row
	if len(fileInfos) == 0 {
		r = stmt.QueryRow(bid, pInfo.ID, pInfo.Date, pInfo.MessageID,
			pInfo.Title, pInfo.Author, pInfo.Trip, pInfo.Message)
	} else {
		x := 8
		args := make([]interface{}, x+(len(fileInfos)*3))
		args[0] = bid
		args[1] = pInfo.ID
		args[2] = pInfo.Date
		args[3] = pInfo.MessageID
		args[4] = pInfo.Title
		args[5] = pInfo.Author
		args[6] = pInfo.Trip
		args[7] = pInfo.Message
		for i := range fileInfos {
			args[x+0] = fileInfos[i].ID
			args[x+1] = fileInfos[i].Thumb
			args[x+2] = fileInfos[i].Original
			x += 3
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
