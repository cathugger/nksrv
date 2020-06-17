package pipostbase

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"

	. "nksrv/lib/logx"
	"nksrv/lib/mailib"
	"nksrv/lib/psqlib/internal/pibase"
)

const postTQMsgArgCount = 16
const postTQFileArgCount = 8

func getNTStmt(sp *pibase.PSQLIB, n int) (s *sql.Stmt, err error) {
	sp.NTMutex.RLock()
	s = sp.NTStmts[n]
	sp.NTMutex.RUnlock()

	if s != nil {
		return
	}

	sp.NTMutex.Lock()
	defer sp.NTMutex.Unlock()

	// there couldve been race so re-examine situation
	s = sp.NTStmts[n]
	if s != nil {
		return
	}
	// confirmed no statement is there yet.
	// create new
	var st string
	// header
	sth := `WITH
	ugp AS (
		INSERT INTO
			ib0.gposts
			(
				date_sent,     -- 1
				date_recv,     -- NOW()
				f_count,       -- 2
				msgid,         -- 3
				title,         -- 4
				author,        -- 5
				trip,          -- 6
				message,       -- 7
				headers,       -- 8
				attrib,        -- 9
				layout,        -- 10
				extras         -- 11
			)
		VALUES
			(
				$1,        -- date_sent
				NOW(),     -- date_recv
				$2,        -- f_count
				$3,        -- msgid
				$4,        -- title
				$5,        -- author
				$6,        -- trip
				$7,        -- message
				$8,        -- headers
				$9,        -- attrib
				$10,       -- layout
				$11        -- extras
			)
		RETURNING
			g_p_id,
			date_sent,
			date_recv,
			f_count
	),
	ut AS (
		INSERT INTO
			ib0.threads (
				b_id,
				g_t_id,
				b_t_name,
				bump,
				skip_over
			)
		SELECT
			$12,        -- b_id
			ugp.g_p_id, -- g_t_id
			$13,        -- b_t_name
			$1,         -- date_sent
			$14         -- skip_over
		FROM
			ugp
		RETURNING
			b_t_id
	),
	ubp AS (
		INSERT INTO
			ib0.bposts (
				b_id,
				b_t_id,
				b_p_id,
				p_name,
				g_p_id,
				msgid,
				date_sent,
				date_recv,
				sage,
				f_count,
				mod_id,
				attrib
			)
		SELECT
			$12,           -- b_id
			ut.b_t_id,     -- b_t_id
			ut.b_t_id,     -- b_p_id
			$13,           -- p_name
			ugp.g_p_id,    -- g_p_id
			$3,            -- msgid
			ugp.date_sent, -- date_sent
			ugp.date_recv, -- date_recv
			FALSE,         -- sage
			ugp.f_count,   -- f_count
			$15,           -- mod_id
			$16            -- attrib
		FROM
			ut
		CROSS JOIN
			ugp
		RETURNING
			g_p_id,
			b_p_id
	)`
	// footer
	stf := `
SELECT
	g_p_id,
	b_p_id
FROM
	ubp`

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
				oname,
				filecfg,
				thumbcfg,
				extras
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
			xq := "($%d::ftype_t,$%d::BIGINT,$%d,$%d,$%d,$%d::jsonb,$%d::jsonb,$%d::jsonb)"
			fmt.Fprintf(&b, xq,
				x+0, x+1, x+2, x+3, x+4, x+5, x+6, x+7)
			x += postTQFileArgCount
		}

		st2 := `
		) AS q1
	)` + stf
		b.WriteString(st2)

		st = b.String()
	}

	//sp.log.LogPrintf(DEBUG, "will prepare newthread(%d) statement:\n%s\n", n, st)
	sp.Log.LogPrintf(DEBUG, "will prepare newthread(%d) statement", n)
	s, err = sp.DB.DB.Prepare(st)
	if err != nil {
		return nil, sp.SQLError("newthread statement preparation", err)
	}
	sp.Log.LogPrintf(DEBUG, "newthread(%d) statement prepared successfully", n)

	sp.NTStmts[n] = s
	return
}

func insertNewThread(
	sp *pibase.PSQLIB,
	tx *sql.Tx, gstmt *sql.Stmt,
	bid boardID, pInfo mailib.PostInfo, skipover bool, modid uint64) (
	gpid postID, bpid postID, duplicate bool, err error) {

	if len(pInfo.H) == 0 {
		panic("post should have header filled")
	}

	stmt := tx.Stmt(gstmt)

	Hjson := MustMarshal(pInfo.H)
	GAjson := MustMarshal(pInfo.GA)
	Ljson := MustMarshal(&pInfo.L)
	Ejson := MustMarshal(&pInfo.E)
	BAjson := MustMarshal(pInfo.BA)

	smodid := sql.NullInt64{Int64: int64(modid), Valid: modid != 0}

	sp.Log.LogPrintf(DEBUG, "NEWTHREAD %s start <%s>", pInfo.ID, pInfo.MessageID)

	var r *sql.Row
	if len(pInfo.FI) == 0 {
		r = stmt.QueryRow(
			pInfo.Date,
			pInfo.FC,
			pInfo.MessageID,
			pInfo.MI.Title,
			pInfo.MI.Author,
			pInfo.MI.Trip,
			pInfo.MI.Message,
			Hjson,
			GAjson,
			Ljson,
			Ejson,

			bid,
			pInfo.ID,
			skipover,
			smodid,
			BAjson)
	} else {

		x := postTQMsgArgCount
		xf := postTQFileArgCount
		args := make([]interface{}, x+(len(pInfo.FI)*xf))

		args[0] = pInfo.Date
		args[1] = pInfo.FC
		args[2] = pInfo.MessageID
		args[3] = pInfo.MI.Title
		args[4] = pInfo.MI.Author
		args[5] = pInfo.MI.Trip
		args[6] = pInfo.MI.Message
		args[7] = Hjson
		args[8] = GAjson
		args[9] = Ljson
		args[10] = Ejson

		args[11] = bid
		args[12] = pInfo.ID
		args[13] = skipover
		args[14] = smodid
		args[15] = BAjson

		for i := range pInfo.FI {

			FFjson := MustMarshal(pInfo.FI[i].FileAttrib)
			FTjson := MustMarshal(pInfo.FI[i].ThumbAttrib)
			FEjson := MustMarshal(pInfo.FI[i].Extras)

			args[x+0] = pInfo.FI[i].Type.String()
			args[x+1] = pInfo.FI[i].Size
			args[x+2] = pInfo.FI[i].ID
			args[x+3] = pInfo.FI[i].ThumbField
			args[x+4] = pInfo.FI[i].Original
			args[x+5] = FFjson
			args[x+6] = FTjson
			args[x+7] = FEjson

			x += xf
		}
		r = stmt.QueryRow(args...)
	}

	sp.Log.LogPrintf(DEBUG, "NEWTHREAD %s process", pInfo.ID)

	err = r.Scan(&gpid, &bpid)
	if err != nil {
		if pqerr, ok := err.(*pq.Error); ok && pqerr.Code == "23505" {
			// duplicate
			return 0, 0, true, nil
		}
		err = sp.SQLError("newthread insert query scan", err)
		return
	}

	sp.Log.LogPrintf(DEBUG, "NEWTHREAD %s done", pInfo.ID)

	// done
	return
}
