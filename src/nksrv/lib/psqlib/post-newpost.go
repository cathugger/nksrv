package psqlib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"

	. "nksrv/lib/logx"
	"nksrv/lib/mailib"
)

type npTuple struct {
	n    int
	sage bool
}

const postRQMsgArgCount = 18
const postRQFileArgCount = 8

func (sp *PSQLIB) getNPStmt(t npTuple) (s *sql.Stmt, err error) {
	sp.npMutex.RLock()
	s = sp.npStmts[t]
	sp.npMutex.RUnlock()

	if s != nil {
		return
	}

	sp.npMutex.Lock()
	defer sp.npMutex.Unlock()

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
	ugp AS (
		INSERT INTO
			ib0.posts (
				pdate,         -- 1
				padded,        -- NOW()
				sage,          -- 2
				f_count,       -- 3
				msgid,         -- 4
				title,         -- 5
				author,        -- 6
				trip,          -- 7
				message,       -- 8
				headers,       -- 9
				attrib,        -- 10
				layout,        -- 11
				extras         -- 12
			)
		VALUES
			(
				$1,        -- pdate
				NOW(),     -- padded
				$2,        -- sage
				$3,        -- f_count
				$4,        -- msgid
				$5,        -- title
				$6,        -- author
				$7,        -- trip
				$8,        -- message
				$9,        -- headers
				$10,       -- attrib
				$11,       -- layout
				$12        -- extras
			)
		RETURNING
			g_p_id,pdate,padded,sage
	),
	ub AS (
		UPDATE
			ib0.boards
		SET
			last_id = last_id + 1,
			p_count = p_count + 1
		WHERE
			-- TODO insert into multiple boards
			b_id = $13
		RETURNING
			last_id
	),`
	b.WriteString(st1)

	if !t.sage {
		// bump algo:
		// sages are still counted against bump limit
		// (currently, idk if ok) OP is counted against bump limit
		st_bump := `
	ut AS (
		UPDATE
			ib0.threads
		SET
			bump = pdate,
			p_count = p_count + 1,
			f_count = f_count + $3
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
						b_id = $13 AND b_t_id = $14
					UNION ALL
					SELECT
						$1,
						last_id,
						FALSE
					FROM
						ub
					ORDER BY
						pdate ASC,
						b_p_id ASC
					LIMIT
						$15
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
			b_id = $13 AND b_t_id = $14
	),`
		b.WriteString(st_bump)
	} else {
		st_nobump := `
	ut AS (
		UPDATE
			ib0.threads
		SET
			p_count = p_count + 1,
			f_count = f_count + $3
			--fr_count = fr_count + (CASE WHEN $3 > 0 THEN 1 ELSE 0)
		WHERE
			b_id = $13 AND b_t_id = $14
	),
	utx AS (
		SELECT
			1
		LIMIT
			$15
	),`
		b.WriteString(st_nobump)
	}

	st2 := `
	ubp AS (
		INSERT INTO
			ib0.bposts (
				b_id,
				b_t_id,
				b_p_id,
				p_name,
				g_p_id,
				msgid,
				pdate,
				padded,
				sage,
				mod_id,
				attrib
			)
		SELECT
			$13,        -- b_id
			$14,        -- b_t_id
			ub.last_id, -- b_p_id
			$16,        -- p_name
			ugp.g_p_id, -- g_p_id
			$4,         -- msgid
			ugp.pdate,  -- pdate
			ugp.padded, -- padded
			ugp.sage,   -- sage
			$17,        -- mod_id
			$18         -- attrib
		FROM
			ub
		CROSS JOIN
			ugp
		RETURNING
			g_p_id,b_p_id
	)`
	b.WriteString(st2)

	if t.n != 0 {
		stf1 := `,
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
		SELECT
			*
		FROM (
			SELECT g_p_id
			FROM ugp
		) AS q0
		CROSS JOIN (
			VALUES `
		b.WriteString(stf1)

		x := postRQMsgArgCount + 1 // counting from 1
		for i := 0; i < t.n; i++ {
			if i != 0 {
				b.WriteString(", ")
			}
			xq := "($%d::ftype_t,$%d::BIGINT,$%d,$%d,$%d,$%d::jsonb,$%d::jsonb,$%d::jsonb)"
			fmt.Fprintf(&b, xq,
				x+0, x+1, x+2, x+3, x+4, x+5, x+6, x+7)
			x += postRQFileArgCount
		}

		// footer
		stf2 := `
		) AS q1
	)`
		b.WriteString(stf2)
	}

	st3 := `
SELECT
	g_p_id,b_p_id
FROM
	ubp`

	b.WriteString(st3)

	st := b.String()

	//sp.log.LogPrintf(DEBUG, "will prepare newreply(%d,%t) statement:\n%s\n", t.n, t.sage, st)
	sp.log.LogPrintf(DEBUG, "will prepare newreply(%d,%t) statement", t.n, t.sage)
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
	tx *sql.Tx, gstmt *sql.Stmt,
	rti replyTargetInfo, pInfo mailib.PostInfo, modid int64) (
	gpid postID, bpid postID, duplicate bool, err error) {

	if len(pInfo.H) == 0 {
		panic("post should have header filled")
	}

	stmt := tx.Stmt(gstmt)

	Hjson, err := json.Marshal(pInfo.H)
	if err != nil {
		panic(err)
	}
	GAjson, err := json.Marshal(pInfo.GA)
	if err != nil {
		panic(err)
	}
	Ljson, err := json.Marshal(&pInfo.L)
	if err != nil {
		panic(err)
	}
	Ejson, err := json.Marshal(&pInfo.E)
	if err != nil {
		panic(err)
	}
	BAjson, err := json.Marshal(pInfo.BA)
	if err != nil {
		panic(err)
	}

	smodid := sql.NullInt64{Int64: modid, Valid: modid != 0}

	sp.log.LogPrintf(DEBUG, "NEWPOST %s start", pInfo.ID)

	var r *sql.Row
	if len(pInfo.FI) == 0 {
		r = stmt.QueryRow(
			pInfo.Date,
			pInfo.MI.Sage,
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

			rti.bid,
			rti.tid,
			rti.bumpLimit,

			pInfo.ID,
			smodid,
			BAjson)
	} else {

		x := postRQMsgArgCount
		xf := postRQFileArgCount
		args := make([]interface{}, x+(len(pInfo.FI)*xf))

		args[0] = pInfo.Date
		args[1] = pInfo.MI.Sage
		args[2] = pInfo.FC
		args[3] = pInfo.MessageID
		args[4] = pInfo.MI.Title
		args[5] = pInfo.MI.Author
		args[6] = pInfo.MI.Trip
		args[7] = pInfo.MI.Message
		args[8] = Hjson
		args[9] = GAjson
		args[10] = Ljson
		args[11] = Ejson

		args[12] = rti.bid
		args[13] = rti.tid
		args[14] = rti.bumpLimit

		args[15] = pInfo.ID
		args[16] = smodid
		args[17] = BAjson

		for i := range pInfo.FI {

			FFjson, err := json.Marshal(pInfo.FI[i].FileAttrib)
			if err != nil {
				panic(err)
			}
			FTjson, err := json.Marshal(pInfo.FI[i].ThumbAttrib)
			if err != nil {
				panic(err)
			}
			FEjson, err := json.Marshal(pInfo.FI[i].Extras)
			if err != nil {
				panic(err)
			}

			args[x+0] = pInfo.FI[i].Type.String()
			args[x+1] = pInfo.FI[i].Size
			args[x+2] = pInfo.FI[i].ID
			args[x+3] = pInfo.FI[i].Thumb
			args[x+4] = pInfo.FI[i].Original
			args[x+5] = FFjson
			args[x+6] = FTjson
			args[x+7] = FEjson

			x += xf
		}
		r = stmt.QueryRow(args...)
	}

	sp.log.LogPrintf(DEBUG, "NEWPOST %s process", pInfo.ID)

	err = r.Scan(&gpid, &bpid)
	if err != nil {
		if pqerr, ok := err.(*pq.Error); ok && pqerr.Code == "23505" {
			// duplicate
			return 0, 0, true, nil
		}
		err = sp.sqlError("newreply insert query scan", err)
		return
	}

	sp.log.LogPrintf(DEBUG, "NEWPOST %s done", pInfo.ID)

	// done
	return
}
