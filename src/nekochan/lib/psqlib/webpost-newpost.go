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

const postRQMsgArgCount = 13
const postRQFileArgCount = 5

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
		WHERE bid = $2
		RETURNING lastid
	),`
	b.WriteString(st1)

	if !t.sage {
		st_bump := `
	ut AS (
		UPDATE ib0.threads
		SET bump = pdate
		FROM (
			SELECT pdate
			FROM (
				SELECT pdate,pid,sage
				FROM ib0.posts
				WHERE bid = $2 AND tid = $3 -- count sages against bump limit. because others do it like that :<
				UNION ALL
				SELECT $4,lastid,FALSE
				FROM ub
				ORDER BY pdate ASC,pid ASC
				LIMIT $1
				-- take bump posts, sorted by original date, only upto bump limit
			) AS tt
			WHERE sage != TRUE
			ORDER BY pdate DESC,pid DESC
			LIMIT 1
			-- and pick latest one
		) as xbump
		WHERE bid = $2 AND tid = $3
	),`
		b.WriteString(st_bump)
	}

	st2 := `
	up AS (
		INSERT INTO ib0.posts (bid,tid,pid,pdate,padded,sage,pname,msgid,title,author,trip,message,headers,layout)
		SELECT $2,$3,lastid,$4,NOW(),$5,$6,$7,$8,$9,$10,$11,$12,$13
		FROM ub
		RETURNING pid
	)`
	b.WriteString(st2)

	if t.n != 0 {
		stf1 := `,
	uf AS (
		INSERT INTO ib0.files (bid,pid,ftype,fsize,fname,thumb,oname)
		SELECT *
		FROM (
			SELECT $1,pid
			FROM up
		) AS q0
		CROSS JOIN (
			VALUES `
		b.WriteString(stf1)

		x := postRQMsgArgCount + 1 // counting from 1
		for i := 0; i < t.n; i++ {
			if i != 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "($%d,$%d::BIGINT,$%d,$%d,$%d)", x+0, x+1, x+2, x+3, x+4)
			x += postRQFileArgCount
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

	sp.log.LogPrintf(DEBUG, "will prepare newreply(%d,%t) statement:\n%s\n", t.n, t.sage, st)
	s, err = sp.db.DB.Prepare(st)
	if err != nil {
		return nil, sp.sqlError("newreply statement preparation", err)
	}
	sp.log.LogPrintf(DEBUG, "newreply(%d,%t) statement prepared successfully", t.n, t.sage)

	sp.npStmts[t] = s
	return
}

type replyTargetInfo struct {
	bid       boardID
	tid       postID
	bumpLimit uint32
}

func (sp *PSQLIB) insertNewReply(
	rti replyTargetInfo, pInfo postInfo) (pid postID, err error) {

	if len(pInfo.H) == 0 {
		panic("post should have header filled")
	}

	stmt, err := sp.getNPStmt(npTuple{len(pInfo.FI), pInfo.MI.Sage})
	if err != nil {
		return
	}

	var r *sql.Row
	if len(pInfo.FI) == 0 {
		r = stmt.QueryRow(
			rti.bumpLimit, rti.bid, rti.tid, pInfo.Date, pInfo.MI.Sage,
			pInfo.ID, pInfo.MessageID,
			pInfo.MI.Title, pInfo.MI.Author, pInfo.MI.Trip, pInfo.MI.Message)
	} else {
		x := postRQMsgArgCount
		xf := postRQFileArgCount
		args := make([]interface{}, x+(len(pInfo.FI)*xf))
		args[0] = rti.bumpLimit
		args[1] = rti.bid
		args[2] = rti.tid
		args[3] = pInfo.Date
		args[4] = pInfo.MI.Sage
		args[5] = pInfo.ID
		args[6] = pInfo.MessageID
		args[7] = pInfo.MI.Title
		args[8] = pInfo.MI.Author
		args[9] = pInfo.MI.Trip
		args[10] = pInfo.MI.Message
		args[11] = pInfo.H
		args[12] = &pInfo.L
		for i := range pInfo.FI {
			args[x+0] = FTypeS[pInfo.FI[i].Type]
			args[x+1] = pInfo.FI[i].Size
			args[x+2] = pInfo.FI[i].ID
			args[x+3] = pInfo.FI[i].Thumb
			args[x+4] = pInfo.FI[i].Original
			x += xf
		}
		r = stmt.QueryRow(args...)
	}
	err = r.Scan(&pid)
	if err != nil {
		return 0, sp.sqlError("newreply insert query scan", err)
	}

	// done
	return
}
