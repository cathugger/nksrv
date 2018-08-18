package psqlib

import (
	"database/sql"
	"fmt"
	"strings"

	. "nekochan/lib/logx"
)

type npTuple struct {
	n    int
	sage bool
}

func (sp *PSQLIB) getNPStmt(t npTuple) (s *sql.Stmt, err error) {
	sp.ntMutex.RLock()
	s = sp.npStmts[t]
	sp.ntMutex.RUnlock()

	if s != nil {
		return
	}

	sp.ntMutex.Lock()
	defer sp.ntMutex.Unlock()

	// there couldve been race so re-examine situation
	s = sp.npStmts[t]
	if s != nil {
		return
	}
	// confirmed no statement is there yet.
	// create new

	var b strings.Builder

	// head
	st1 := `WITH
	ub AS (
		UPDATE ib0.boards
		SET lastid = lastid+1
		WHERE bid = $1
		RETURNING lastid
	),`
	b.WriteString(st1)

	if !t.sage {
		st_bump := `
	ut AS (
		UPDATE ib0.threads
		SET bump = $3
		WHERE bid = $1 AND tid = $2 AND bump < $3
	),`
		b.WriteString(st_bump)
	}

	st2 := `
	up AS (
		INSERT INTO ib0.posts (bid,tid,pid,pdate,pname,msgid,title,author,trip,message)
		SELECT $1,$2,lastid,$3,$4,$5,$6,$7,$8,$9
		FROM ub
		RETURNING pid
	)`
	b.WriteString(st2)

	if t.n != 0 {
		stf1 := `,
	uf AS (
		INSERT INTO ib0.files (bid,pid,fname,thumb,oname)
		SELECT *
		FROM (
			SELECT $1,pid
			FROM up
		) AS q0
		CROSS JOIN (
			VALUES `
		b.WriteString(stf1)

		x := 8 // 7 args already, counting from 1
		for i := 0; i < t.n; i++ {
			if i != 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "($%d, $%d, $%d)", x+0, x+1, x+2)
			x += 3
		}

		// footer
		stf2 := `
		) AS q1
	)`
		b.WriteString(stf2)
	}

	st3 := `
SELECT * FROM up`
	b.WriteString(st3)

	st := b.String()

	sp.log.LogPrintf(DEBUG, "will prepare newthread(%d,%t) statement:\n%s\n", t.n, t.sage, st)
	s, err = sp.db.DB.Prepare(st)
	if err != nil {
		return nil, sp.sqlError("newthread statement preparation", err)
	}
	sp.log.LogPrintf(DEBUG, "newthread(%d,%t) statement prepared successfully", t.n, t.sage)

	sp.npStmts[t] = s
	return
}

func (sp *PSQLIB) insertNewReply(bid boardID, tid postID, pInfo postInfo,
	fileInfos []fileInfo) (pid postID, err error) {

	stmt, err := sp.getNPStmt(npTuple{len(fileInfos), pInfo.Sage})
	if err != nil {
		return
	}

	var r *sql.Row
	if len(fileInfos) == 0 {
		r = stmt.QueryRow(bid, tid, pInfo.Date, pInfo.ID, pInfo.MessageID,
			pInfo.Title, pInfo.Author, pInfo.Trip, pInfo.Message)
	} else {
		args := make([]interface{}, 9+(len(fileInfos)*3))
		args[0] = bid
		args[1] = tid
		args[2] = pInfo.Date
		args[3] = pInfo.ID
		args[4] = pInfo.MessageID
		args[5] = pInfo.Title
		args[6] = pInfo.Author
		args[7] = pInfo.Trip
		args[8] = pInfo.Message
		x := 9
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
		return 0, sp.sqlError("newpost insert query scan", err)
	}

	// done
	return
}
