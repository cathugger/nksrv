package psqlib

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/lib/pq"

	xtypes "github.com/jmoiron/sqlx/types"

	"nksrv/lib/ibref_nntp"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	ib0 "nksrv/lib/webib0"
)

// PostgreSQL doesn't wanna optimize LIKE operations at all when used
// with arrays or left joins...
// So generate our own queries.

const selhead_a = `SELECT
	%d`

const selhead_b = `
FROM
	ib0.bposts AS xbp
JOIN
	ib0.threads xt
ON
	xbp.b_id = xt.b_id AND xbp.b_t_id = xt.b_t_id
JOIN
	ib0.boards xb
ON
	xbp.b_id = xb.b_id
JOIN
	ib0.gposts AS xp
ON
	xbp.g_p_id = xp.g_p_id`

const selhead = selhead_a + `,
	'<' || xp.msgid || '>'` + selhead_b

const selhead2 = selhead_a + `,
	xb.b_id,
	xb.b_name,
	xt.b_t_id,
	xt.b_t_name,
	xbp.p_name` + selhead_b

func escapeSQLString(s string) string {
	return strings.Replace(s, "'", "''", -1)
}

func buildMsgIDArray(prefs []string) string {
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
	QueryRow(query string, args ...interface{}) *sql.Row
}

func (sp *PSQLIB) processReferencesOnPost(qq queryable,
	srefs []ibref_nntp.Reference,
	bid boardID, tid postID, isctl bool) (
	inreplyto []string, err error) {

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

	addirt := func(x string) {
		for _, s := range inreplyto {
			if s == x {
				// duplicate
				return
			}
		}
		inreplyto = append(inreplyto, x)
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
	(xbp.b_id = %d AND xbp.b_t_id = %d) DESC,
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

			// if posting onto ctl group, <msgid> references are usually for deletes
			if !isctl {

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

			}

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

		// do query
		rows, err = qq.Query(q)
		if err != nil {
			err = sp.sqlError("references query", err)
			return
		}
		defer rows.Close()
	}

	var r_id int
	var r_msgid string

	for i := range srefs {

		fetchrow := func() (err error) {
			// there's more questions than answers so no need to use `for`
			if r_id <= i && rows.Next() {
				err = rows.Scan(&r_id, &r_msgid)
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

			if r_id == i+1 {
				addirt(r_msgid)
				sp.log.LogPrintf(DEBUG, "ref: %#v %q", srefs[i], r_msgid)
			}

		} else if len(srefs[i].Board) != 0 {

			sp.log.LogPrintf(DEBUG, "bref: %#v", srefs[i])

		} else if len(srefs[i].MsgID) != 0 {

			err = fetchrow()
			if err != nil {
				return
			}

			if r_id == i+1 {
				addirt(r_msgid)
				sp.log.LogPrintf(DEBUG, "mref: %#v %q", srefs[i], r_msgid)
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

func (sp *PSQLIB) processReferencesOnIncoming(qq queryable,
	srefs []ibref_nntp.Reference, irefs []ibref_nntp.Index,
	prefs []string,
	bid boardID, tid postID) (
	arefs []ib0.IBMessageReference,
	err error) {

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
						Start: uint(irefs[i].Start),
						End:   uint(irefs[i].End),
					}
					r.Post = r_pname

					if r_bid != bid {
						r.Board = r_bname
						r.Thread = r_tname
					} else if r_tid != tid {
						r.Thread = r_tname
					}

					arefs = append(arefs, r)
				}

			} else {

				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(irefs[i].Start),
						End:   uint(irefs[i].End),
					}
					r.Board = r_bname
					r.Thread = r_tname
					r.Post = r_pname

					arefs = append(arefs, r)
				}

			}
		} else if len(srefs[i].Board) != 0 {

			// plain board - don't need SQL, just take in

			r := ib0.IBMessageReference{
				Start: uint(irefs[i].Start),
				End:   uint(irefs[i].End),
			}
			r.Board = string(srefs[i].Board)

			arefs = append(arefs, r)

		} else if len(srefs[i].MsgID) != 0 {

			err = fetchrow()
			if err != nil {
				return
			}

			if r_id == i+1 {
				r := ib0.IBMessageReference{
					Start: uint(irefs[i].Start),
					End:   uint(irefs[i].End),
				}
				r.Board = r_bname
				r.Thread = r_tname
				r.Post = r_pname

				arefs = append(arefs, r)
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

func (sp *PSQLIB) insertXRefs(
	st *sql.Stmt, bid boardID, bpid postID, xrefs []ibref_nntp.Reference) (err error) {

	if len(xrefs) == 0 {
		// don't waste resources if we have no refs
		// we insert them only once, and they won't need change
		return
	}

	postids := make([]sql.NullString, len(xrefs))
	boards := make([]sql.NullString, len(xrefs))
	msgids := make([]sql.NullString, len(xrefs))

	for i := range xrefs {
		if xrefs[i].Post != "" {
			postids[i].Valid = true
			postids[i].String = xrefs[i].Post
			if xrefs[i].Board != "" {
				boards[i].Valid = true
				boards[i].String = xrefs[i].Board
			}
		} else if xrefs[i].Board != "" {
			boards[i].Valid = true
			boards[i].String = xrefs[i].Board
		} else if xrefs[i].MsgID != "" {
			msgids[i].Valid = true
			msgids[i].String = xrefs[i].MsgID
		}
	}

	_, err = st.Exec(
		bid, bpid,
		pq.Array(postids),
		pq.Array(boards),
		pq.Array(msgids))
	if err != nil {
		err = sp.sqlError("xrefs insert exec", err)
		return
	}

	return
}

type xRefData struct {
	b_id       boardID
	b_p_id     postID
	message    string
	inreplyto  []string
	activ_refs []ib0.IBMessageReference
	b_t_id     postID
}

func (sp *PSQLIB) findReferences(
	st *sql.Stmt, off_b boardID, off_b_p postID,
	pname string, pboard string, msgid TCoreMsgIDStr) (
	xrefs []xRefData, err error) {

	rows, err := st.Query(off_b, off_b_p, pname, pboard, string(msgid))
	if err != nil {
		err = sp.sqlError("mod_ref_find_post query", err)
		return
	}

	for rows.Next() {
		var b_id boardID
		var b_p_id postID
		var msg string
		var inreplyto sql.NullString
		var j_activ_refs xtypes.JSONText
		var b_t_id postID

		err = rows.Scan(
			&b_id, &b_p_id, &msg, &inreplyto, &j_activ_refs, &b_t_id)
		if err != nil {
			rows.Close()
			err = sp.sqlError("mod_ref_find_post query rows scan", err)
			return
		}

		var activ_refs []ib0.IBMessageReference
		err = j_activ_refs.Unmarshal(&activ_refs)
		if err != nil {
			rows.Close()
			err = sp.sqlError("mod_ref_find_post json unmarshal", err)
			return
		}

		xrefs = append(xrefs, xRefData{
			b_id:       b_id,
			b_p_id:     b_p_id,
			message:    msg,
			inreplyto:  mail.ExtractAllValidReferences(nil, inreplyto.String),
			activ_refs: activ_refs,
			b_t_id:     b_t_id,
		})
	}
	if err = rows.Err(); err != nil {
		err = sp.sqlError("mod_ref_find_post query rows", err)
		return
	}

	return
}

func (sp *PSQLIB) updatePostReferences(
	st *sql.Stmt, b_id boardID, b_p_id postID,
	activ_refs []ib0.IBMessageReference) (
	err error) {

	Ajson, err := json.Marshal(activ_refs)
	if err != nil {
		panic(err)
	}

	_, err = st.Exec(b_id, b_p_id, Ajson)
	if err != nil {
		return sp.sqlError("mod_update_bpost_activ_refs exec", err)
	}

	return
}

func (sp *PSQLIB) processRefsAfterPost(
	tx *sql.Tx,
	srefs []ibref_nntp.Reference, irefs []ibref_nntp.Index,
	prefs []string,
	b_id boardID, b_t_id, b_p_id postID,
	postid, newsgroup string, msgid TCoreMsgIDStr) (err error) {

	// write our declaration of references
	xref_wr_st := tx.Stmt(sp.st_prep[st_mod_ref_write])
	err = sp.insertXRefs(xref_wr_st, b_id, b_p_id, srefs)
	if err != nil {
		return
	}
	// process our stuff
	arefs, err := sp.processReferencesOnIncoming(
		tx, srefs, irefs, prefs, b_id, b_t_id)
	if err != nil {
		return
	}
	// then, put proper references in our bpost
	xref_up_st := tx.Stmt(sp.st_prep[st_mod_update_bpost_activ_refs])
	err = sp.updatePostReferences(
		xref_up_st,
		b_id, b_p_id,
		arefs)
	if err != nil {
		return
	}
	// then, fixup any other stuff possibly referring to us
	err = sp.fixupAffectedXRefsInTx(tx, postid, newsgroup, msgid, xref_up_st)
	if err != nil {
		return
	}

	return
}

func (sp *PSQLIB) fixupAffectedXRefsInTx(
	tx *sql.Tx, p_name, b_name string, msgid TCoreMsgIDStr,
	xref_up_st *sql.Stmt) (err error) {

	xref_fn_st := tx.Stmt(sp.st_prep[st_mod_ref_find_post])

	if p_name == "" || b_name == "" || msgid == "" {
		panic("wtf")
	}

	var xrefpostsinfos []xRefData
	// in loop because can repeat
	off_b, off_b_p := boardID(0), postID(0)
	for {
		// obtain infos about posts with refs
		xrefpostsinfos, err = sp.findReferences(
			xref_fn_st, off_b, off_b_p, p_name, b_name, msgid)
		if err != nil {
			return
		}

		if len(xrefpostsinfos) != 0 {
			sp.log.LogPrintf(
				DEBUG, "found %d revref posts", len(xrefpostsinfos))
		}

		for i := range xrefpostsinfos {

			var arefs []ib0.IBMessageReference

			// update references and collect new failed references
			srefs, irefs :=
				ibref_nntp.ParseReferences(xrefpostsinfos[i].message)
			arefs, err = sp.processReferencesOnIncoming(
				tx,
				srefs, irefs,
				xrefpostsinfos[i].inreplyto,
				xrefpostsinfos[i].b_id, xrefpostsinfos[i].b_t_id)
			if err != nil {
				return
			}

			if reflect.DeepEqual(
				xrefpostsinfos[i].activ_refs, arefs) {

				sp.log.LogPrintf(
					DEBUG, "fixupAffectedXRefsInTx %d: unchanged", i)
				continue
			}

			sp.log.LogPrintf(
				DEBUG,
				"fixupAffectedXRefsInTx %d: %d arefs %d srefs",
				i,
				len(arefs),
				len(srefs))

			// store updated refs
			err = sp.updatePostReferences(
				xref_up_st,
				xrefpostsinfos[i].b_id, xrefpostsinfos[i].b_p_id,
				arefs)
			if err != nil {
				return
			}
		}

		if len(xrefpostsinfos) < 5000 {

			// the usual case
			break

		} else {

			// we can't know if we processed them all, so lets do search again
			off_b, off_b_p =
				xrefpostsinfos[len(xrefpostsinfos)-1].b_id,
				xrefpostsinfos[len(xrefpostsinfos)-1].b_p_id

			sp.log.LogPrintf(DEBUG, "continuing failref loop")

			continue

		}
	}

	return
}
