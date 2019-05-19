package psqlib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"

	xtypes "github.com/jmoiron/sqlx/types"

	"centpd/lib/ibref_nntp"
	. "centpd/lib/logx"
	"centpd/lib/mail"
	mm "centpd/lib/minimail"
	ib0 "centpd/lib/webib0"
)

// PostgreSQL doesn't wanna optimize LIKE operations at all when used
// with arrays or left joins...
// So generate our own queries.

const selhead_a = `SELECT
	%d,
	xb.b_id,
	xb.b_name,
	xt.t_id,
	xt.t_name,
	xbp.p_name`

const selhead_b = `
FROM
	ib0.bposts AS xbp
JOIN
	ib0.threads xt
ON
	xbp.b_id = xt.b_id AND xbp.t_id = xt.t_id
JOIN
	ib0.boards xb
ON
	xbp.b_id = xb.b_id
JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id`

const selhead = selhead_a + `,
	'<' || xp.msgid || '>'` + selhead_b

const selhead2 = selhead_a + selhead_b

func escapeSQLString(s string) string {
	return strings.Replace(s, "'", "''", -1)
}

func buildMsgIDArray(prefs []mm.FullMsgIDStr) string {
	var b strings.Builder

	b.WriteString("ARRAY[")
	for i := range prefs {
		if i != 0 {
			b.WriteString("','")
		} else {
			b.WriteByte('\'')
		}
		b.WriteString(escapeSQLString(string(prefs[i])))
	}
	if len(prefs) != 0 {
		b.WriteByte('\'')
	}
	b.WriteByte(']')

	return b.String()
}

type queryable interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

func (sp *PSQLIB) processReferencesOnPost(
	qq queryable, msg string, bid boardID, tid postID) (
	refs []ib0.IBMessageReference, inreplyto []string,
	failrefs []ibref_nntp.Reference, err error) {

	srefs := ibref_nntp.ParseReferences(msg)

	// build query
	b := &strings.Builder{}

	first := true

	next := func() {
		if b.Len() == 0 {
			b.WriteString("SELECT * FROM\n(\n")
		} else {
			if first {
				b.WriteString("\n) AS meow\nUNION ALL\n(\n")
				first = false
			} else {
				b.WriteString("\n)\nUNION ALL\n(\n")
			}
		}
	}

	for i := range srefs {

		if len(srefs[i].Post) != 0 {

			next()

			if len(srefs[i].Board) == 0 {
				// only postID
				q := selhead + `
WHERE
	xbp.p_name LIKE '%s%%'
ORDER BY
	(xbp.b_id = %d AND xbp.t_id = %d) DESC,
	(xbp.b_id = %d) DESC,
	xbp.g_p_id DESC,
	xb.b_name ASC
LIMIT
	1`
				fmt.Fprintf(b, q, i+1, srefs[i].Post, bid, tid, bid)

			} else {
				// board+postID
				q := selhead + `
WHERE
	xbp.p_name LIKE '%s%%' AND xb.b_name = '%s'
ORDER BY
	xbp.g_p_id DESC
LIMIT
	1`
				fmt.Fprintf(b, q, i+1, srefs[i].Post, srefs[i].Board)

			}
		} else if len(srefs[i].Board) != 0 {
			// board
			// nothing, we don't need to look it up
		} else if len(srefs[i].MsgID) != 0 {
			// message-id

			next()

			q := selhead + `
WHERE
	xp.msgid = '%s'
ORDER BY
	(xbp.b_id = %d) DESC,
	xb.b_name ASC
LIMIT
	1`
			fmt.Fprintf(b, q, i+1, escapeSQLString(srefs[i].MsgID), bid)

		} else {
			panic("wtf")
		}
	}

	var rows *sql.Rows

	if b.Len() != 0 {
		// finish it up
		if first {
			b.WriteString("\n) AS meow")
		} else {
			b.WriteString("\n)")
		}

		q := b.String()

		sp.log.LogPrintf(DEBUG, "SQL for post references:\n%s", q)

		rows, err = qq.Query(q)
		if err != nil {
			err = sp.sqlError("references query", err)
			return
		}
		defer rows.Close()
	}

	var r_id int
	var r_bid boardID
	var r_bname string
	var r_tid postID
	var r_tname string
	var r_pname string
	var r_msgid string

	for i := range srefs {

		fetchrow := func() (err error) {
			if r_id <= i && rows.Next() {
				err = rows.Scan(
					&r_id,
					&r_bid, &r_bname,
					&r_tid, &r_tname,
					&r_pname, &r_msgid)
				if err != nil {
					err = sp.sqlError("references query scan", err)
					return
				}
			}
			return
		}

		if len(srefs[i].Post) != 0 {
			err = fetchrow()
			if err != nil {
				return
			}

			if len(srefs[i].Board) == 0 {
				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(srefs[i].Start),
						End:   uint(srefs[i].End),
					}
					r.Post = r_pname

					if r_bid != bid {
						r.Board = r_bname
						r.Thread = r_tname
					} else if r_tid != tid {
						r.Thread = r_tname
					}
					refs = append(refs, r)
					inreplyto = append(inreplyto, r_msgid)
					sp.log.LogPrintf(DEBUG, "ref: %#v %q", r, r_msgid)
				} else {
					failrefs = append(failrefs, srefs[i].Reference)
				}
			} else {
				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(srefs[i].Start),
						End:   uint(srefs[i].End),
					}
					r.Board = r_bname
					r.Thread = r_tname
					r.Post = r_pname

					refs = append(refs, r)
					inreplyto = append(inreplyto, r_msgid)
					sp.log.LogPrintf(DEBUG, "cref: %#v %q", r, r_msgid)
				} else {
					failrefs = append(failrefs, srefs[i].Reference)
				}
			}
		} else if len(srefs[i].Board) != 0 {

			r := ib0.IBMessageReference{
				Start: uint(srefs[i].Start),
				End:   uint(srefs[i].End),
			}

			r.Board = string(srefs[i].Board)
			refs = append(refs, r)
			sp.log.LogPrintf(DEBUG, "bref: %#v", r)

		} else if len(srefs[i].MsgID) != 0 {

			err = fetchrow()
			if err != nil {
				return
			}

			if r_id == i+1 {
				r := ib0.IBMessageReference{
					Start: uint(srefs[i].Start),
					End:   uint(srefs[i].End),
				}
				r.Board = r_bname
				r.Thread = r_tname
				r.Post = r_pname

				refs = append(refs, r)
				inreplyto = append(inreplyto, r_msgid)
				sp.log.LogPrintf(DEBUG, "mref: %#v %q", r, r_msgid)
			} else {
				failrefs = append(failrefs, srefs[i].Reference)
			}
		} else {
			panic("wtf")
		}
	}
	if rows != nil {
		err = rows.Err()
		if err != nil {
			err = sp.sqlError("references query rows", err)
			return
		}
	}

	return
}

func (sp *PSQLIB) processReferencesOnIncoming(
	qq queryable, msg string, prefs []mm.FullMsgIDStr,
	bid boardID, tid postID) (
	refs []ib0.IBMessageReference, failrefs []ibref_nntp.Reference,
	err error) {

	srefs := ibref_nntp.ParseReferences(msg)

	if len(srefs) == 0 {
		return
	}

	qprefs := buildMsgIDArray(prefs)

	// build query
	b := &strings.Builder{}

	first := true

	next := func() {
		if b.Len() == 0 {
			b.WriteString("SELECT * FROM\n(\n")
		} else {
			if first {
				b.WriteString("\n) AS meow\nUNION ALL\n(\n")
				first = false
			} else {
				b.WriteString("\n)\nUNION ALL\n(\n")
			}
		}
	}

	for i := range srefs {

		if len(srefs[i].Post) != 0 {

			next()

			if len(srefs[i].Board) == 0 {
				// only postID
				q := selhead2 + `
WHERE
	xbp.p_name LIKE '%s%%'
ORDER BY
	('<' || xp.msgid || '>' = ANY(%s::text[])) DESC,
	(xbp.b_id = %d) DESC,
	xbp.g_p_id ASC,
	xb.b_name ASC
LIMIT
	1`
				fmt.Fprintf(b, q, i+1, srefs[i].Post, qprefs, bid)

			} else {
				// board+postID
				q := selhead2 + `
WHERE
	xbp.p_name LIKE '%s%%' AND xb.b_name = '%s'
ORDER BY
	('<' || xp.msgid || '>' = ANY(%s::text[])) DESC,
	xbp.g_p_id ASC
LIMIT
	1`
				fmt.Fprintf(b, q, i+1, srefs[i].Post, srefs[i].Board, qprefs)

			}
		} else if len(srefs[i].Board) != 0 {
			// board
			// nothing, we don't need to look it up
		} else if len(srefs[i].MsgID) != 0 {
			// msgid

			next()

			q := selhead2 + `
WHERE
	xp.msgid = '%s'
ORDER BY
	(xbp.b_id = %d) DESC,
	xb.b_name ASC
LIMIT
	1`
			fmt.Fprintf(b, q, i+1, escapeSQLString(srefs[i].MsgID), bid)

		} else {
			panic("wtf")
		}
	}

	var rows *sql.Rows

	if b.Len() != 0 {
		// finish it up
		if first {
			b.WriteString("\n) AS meow")
		} else {
			b.WriteString("\n)")
		}

		rows, err = qq.Query(b.String())
		if err != nil {
			err = sp.sqlError("references query", err)
			return
		}
		defer rows.Close()
	}

	var r_id int
	var r_bid boardID
	var r_bname string
	var r_tid postID
	var r_tname string
	var r_pname string

	for i := range srefs {

		fetchrow := func() (err error) {
			if r_id <= i && rows.Next() {
				err = rows.Scan(
					&r_id,
					&r_bid, &r_bname,
					&r_tid, &r_tname,
					&r_pname)
				if err != nil {
					err = sp.sqlError("references query scan", err)
					return
				}
			}
			return
		}

		if len(srefs[i].Post) != 0 {

			err = fetchrow()
			if err != nil {
				return
			}

			if len(srefs[i].Board) == 0 {
				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(srefs[i].Start),
						End:   uint(srefs[i].End),
					}
					r.Post = r_pname

					if r_bid != bid {
						r.Board = r_bname
						r.Thread = r_tname
					} else if r_tid != tid {
						r.Thread = r_tname
					}

					refs = append(refs, r)
				} else {
					failrefs = append(failrefs, srefs[i].Reference)
				}
			} else {
				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(srefs[i].Start),
						End:   uint(srefs[i].End),
					}
					r.Board = r_bname
					r.Thread = r_tname
					r.Post = r_pname

					refs = append(refs, r)
				} else {
					failrefs = append(failrefs, srefs[i].Reference)
				}
			}
		} else if len(srefs[i].Board) != 0 {

			// plain board - don't need SQL, just take in

			r := ib0.IBMessageReference{
				Start: uint(srefs[i].Start),
				End:   uint(srefs[i].End),
			}
			r.Board = string(srefs[i].Board)

			refs = append(refs, r)

		} else if len(srefs[i].MsgID) != 0 {

			err = fetchrow()
			if err != nil {
				return
			}

			if r_id == i+1 {
				r := ib0.IBMessageReference{
					Start: uint(srefs[i].Start),
					End:   uint(srefs[i].End),
				}
				r.Board = r_bname
				r.Thread = r_tname
				r.Post = r_pname

				refs = append(refs, r)
			} else {
				failrefs = append(failrefs, srefs[i].Reference)
			}
		}
	}
	if rows != nil {
		err = rows.Err()
		if err != nil {
			err = sp.sqlError("references query rows", err)
			return
		}
	}

	return
}

func (sp *PSQLIB) writeFailRefs(
	st *sql.Stmt, gpid postID, failref []ibref_nntp.Reference) (err error) {

	postids := make([]sql.NullString, len(failref))
	boards := make([]sql.NullString, len(failref))
	msgids := make([]sql.NullString, len(failref))

	for i := range failref {
		if failref[i].Post != "" {
			postids[i].Valid = true
			postids[i].String = failref[i].Post

			if failref[i].Board != "" {
				boards[i].Valid = true
				boards[i].String = failref[i].Board
			}
		} else if failref[i].MsgID != "" {
			msgids[i].Valid = true
			msgids[i].String = failref[i].MsgID
		}
	}

	_, err = st.Exec(gpid,
		pq.Array(postids), pq.Array(boards), pq.Array(msgids))
	if err != nil {
		err = sp.sqlError("failrefs insert exec", err)
		return
	}

	return
}

func (sp *PSQLIB) writeFailRefsAfterPost(
	st *sql.Stmt, gpid postID, failref []ibref_nntp.Reference) (err error) {

	if len(failref) == 0 {
		// after post, we don't have anything to delete
		return
	}

	return sp.writeFailRefs(st, gpid, failref)
}

type failedRefData struct {
	gpid      postID
	message   string
	inreplyto []FullMsgIDStr
	pattrib   postAttributes
	bid       boardID
	tid       postID
}

func (sp *PSQLIB) findFailedReferences(
	st *sql.Stmt, off postID, pname string, pboard string, msgid CoreMsgIDStr) (
	frefs []failedRefData, err error) {

	rows, err := st.Query(off, pname, pboard, string(msgid))
	if err != nil {
		err = sp.sqlError("find_failrefs query", err)
		return
	}

	for rows.Next() {
		var gpid postID
		var msg string
		var inreplyto sql.NullString
		var j_attrib xtypes.JSONText
		var bid boardID
		var tid postID

		err = rows.Scan(&gpid, &msg, &inreplyto, &j_attrib, &bid, &tid)
		if err != nil {
			rows.Close()
			err = sp.sqlError("find_failrefs query rows scan", err)
			return
		}

		p_attrib := defaultPostAttributes
		err = j_attrib.Unmarshal(&p_attrib)
		if err != nil {
			rows.Close()
			err = sp.sqlError("find_failrefs json unmarshal", err)
			return
		}

		frefs = append(frefs, failedRefData{
			gpid:      gpid,
			message:   msg,
			inreplyto: mail.ExtractAllValidReferences(nil, inreplyto.String),
			pattrib:   p_attrib,
			bid:       bid,
			tid:       tid,
		})
	}
	if err = rows.Err(); err != nil {
		err = sp.sqlError("find_failrefs query rows", err)
		return
	}

	return
}

func (sp *PSQLIB) updatePostReferences(
	st *sql.Stmt, gpid postID, pattrib postAttributes) (err error) {

	Ajson, err := json.Marshal(pattrib)
	if err != nil {
		panic(err)
	}

	_, err = st.Exec(gpid, Ajson)
	if err != nil {
		return sp.sqlError("update_post_refs exec", err)
	}

	return
}

func (sp *PSQLIB) fixupFailRefsInTx(
	tx *sql.Tx, gpid postID, failrefs []ibref_nntp.Reference,
	p_name, b_name string, msgid CoreMsgIDStr) (err error) {

	if p_name == "" || b_name == "" || msgid == "" {
		panic("wtf")
	}

	failref_wr_st := tx.Stmt(sp.st_prep[st_Web_failref_write])
	failref_up_st := tx.Stmt(sp.st_prep[st_Web_update_post_attrs])
	failref_fn_st := tx.Stmt(sp.st_prep[st_Web_failref_find])

	if len(failrefs) != 0 {
		sp.log.LogPrintf(DEBUG, "writing %d failed refs", len(failrefs))
	}

	// put our failed references
	err = sp.writeFailRefsAfterPost(failref_wr_st, gpid, failrefs)
	if err != nil {
		return
	}

	var frefpostsinfos []failedRefData
	// in loop because can repeat
	offset := postID(0)
	for {
		// obtain infos about posts with failed refs we can fix
		frefpostsinfos, err = sp.findFailedReferences(
			failref_fn_st, offset, p_name, b_name, msgid)
		if err != nil {
			return
		}

		if len(frefpostsinfos) != 0 || offset != 0 {
			sp.log.LogPrintf(DEBUG, "found %d failref posts", len(frefpostsinfos))
		}

		for i := range frefpostsinfos {
			var newfailrefs []ibref_nntp.Reference

			// update references and collect new failed references
			frefpostsinfos[i].pattrib.References, newfailrefs, err =
				sp.processReferencesOnIncoming(
					tx, frefpostsinfos[i].message, frefpostsinfos[i].inreplyto,
					frefpostsinfos[i].bid, frefpostsinfos[i].tid)
			if err != nil {
				return
			}

			sp.log.LogPrintf(DEBUG, "failrefpost %d: %d refs %d fails",
				i, len(frefpostsinfos[i].pattrib.References), len(newfailrefs))

			// store updated refs
			err = sp.updatePostReferences(
				failref_up_st, frefpostsinfos[i].gpid, frefpostsinfos[i].pattrib)
			if err != nil {
				return
			}

			// unconditionally store new failed refs
			// also deletes old ones of same gpid
			err = sp.writeFailRefs(
				failref_wr_st, frefpostsinfos[i].gpid, newfailrefs)
			if err != nil {
				return
			}
		}

		if len(frefpostsinfos) < 8192 {
			// the usual case
			break
		} else {
			// we can't know if we processed them all, so lets do search again
			offset = frefpostsinfos[len(frefpostsinfos)-1].gpid
			sp.log.LogPrintf(DEBUG, "continuing failref loop")
			continue
		}
	}

	return
}
