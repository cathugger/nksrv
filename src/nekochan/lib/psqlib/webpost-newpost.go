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
		SET bump = pdate
		FROM (
			SELECT pdate
			FROM (
				SELECT pdate,pid,sage
				FROM ib0.posts
				WHERE bid = $1 AND tid = $2 -- count sages against bump limit. because others do it like that :<
				UNION ALL
				SELECT $3,lastid,FALSE
				FROM ub
				ORDER BY pdate ASC,pid ASC
				LIMIT $11
				-- take bump posts, sorted by original date, only upto bump limit
			) AS tt
			WHERE sage != TRUE
			ORDER BY pdate DESC,pid DESC
			LIMIT 1
			-- and pick latest one
		) as xbump
		WHERE bid = $1 AND tid = $2
	),`
		b.WriteString(st_bump)
	}

	st2 := `
	up AS (
		INSERT INTO ib0.posts (bid,tid,pid,pdate,padded,sage,pname,msgid,title,author,trip,message)
		SELECT $1,$2,lastid,$3,NOW(),$4,$5,$6,$7,$8,$9,$10
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

		x := 12 // 11 args already, counting from 1
		for i := 0; i < t.n; i++ {
			if i != 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "($%d,$%d::BIGINT,$%d,$%d,$%d)", x+0, x+1, x+2, x+3, x+4)
			x += 5
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

func (sp *PSQLIB) insertNewReply(rti replyTargetInfo, pInfo postInfo,
	fileInfos []fileInfo) (pid postID, err error) {

	stmt, err := sp.getNPStmt(npTuple{len(fileInfos), pInfo.Sage})
	if err != nil {
		return
	}

	var r *sql.Row
	if len(fileInfos) == 0 {
		r = stmt.QueryRow(rti.bid, rti.tid, pInfo.Date, pInfo.Sage, pInfo.ID,
			pInfo.MessageID, pInfo.Title, pInfo.Author, pInfo.Trip, pInfo.Message,
			rti.bumpLimit)
	} else {
		x := 11
		xf := 5
		args := make([]interface{}, x+(len(fileInfos)*xf))
		args[0] = rti.bid
		args[1] = rti.tid
		args[2] = pInfo.Date
		args[3] = pInfo.Sage
		args[4] = pInfo.ID
		args[5] = pInfo.MessageID
		args[6] = pInfo.Title
		args[7] = pInfo.Author
		args[8] = pInfo.Trip
		args[9] = pInfo.Message
		args[10] = rti.bumpLimit
		for i := range fileInfos {
			args[x+0] = fileInfos[i].Type
			args[x+1] = fileInfos[i].Size
			args[x+2] = fileInfos[i].ID
			args[x+3] = fileInfos[i].Thumb
			args[x+4] = fileInfos[i].Original
			x += xf
		}
		r = stmt.QueryRow(args...)
	}
	err = r.Scan(&rti.tid)
	if err != nil {
		return 0, sp.sqlError("newreply insert query scan", err)
	}

	// done
	return
}
