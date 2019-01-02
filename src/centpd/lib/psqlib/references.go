package psqlib

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"

	. "centpd/lib/logx"
	mm "centpd/lib/minimail"
	ib0 "centpd/lib/webib0"
)

var re_ref = regexp.MustCompile(
	`>> ?([0-9a-fA-F]{8,40})`)
var re_cref = regexp.MustCompile(
	`>>> ?/([0-9a-zA-Z+_.-]{1,255})/(?: ?([0-9a-fA-F]{8,40}))?`)

type sliceReference struct {
	start, end int
	board      string
	post       string
}

func parseReferences(msg string) (srefs []sliceReference) {
	var sm [][]int
	sm = re_ref.FindAllStringSubmatchIndex(msg, -1)
	for i := range sm {
		srefs = append(srefs, sliceReference{
			start: sm[i][0],
			end:   sm[i][1],
			post:  msg[sm[i][2]:sm[i][3]],
		})
	}
	sm = re_cref.FindAllStringSubmatchIndex(msg, -1)
	for i := range sm {
		x := sliceReference{
			start: sm[i][0],
			end:   sm[i][1],
			board: msg[sm[i][2]:sm[i][3]],
		}
		if sm[i][4] >= 0 {
			x.post = msg[sm[i][4]:sm[i][5]]
		}
		srefs = append(srefs, x)
	}
	// sort by position
	sort.Slice(srefs, func(i, j int) bool {
		return srefs[i].start < srefs[j].start
	})
	// remove overlaps, if any
	for i := 1; i < len(srefs); i++ {
		if srefs[i-1].end > srefs[i].start {
			srefs = append(srefs[:i], srefs[i+1:]...)
			i--
		}
	}
	// limit
	if len(srefs) > 255 {
		srefs = srefs[:255]
	}
	return
}

// PostgreSQL doesn't wanna optimize LIKE operations at all when used
// with arrays or left joins...
// So generate our own queries.

func (sp *PSQLIB) processReferencesOnPost(
	msg string, bid boardID, tid postID) (
	refs []ib0.IBMessageReference, inreplyto []string, err error) {

	srefs := parseReferences(msg)

	// build query
	b := &strings.Builder{}

	next := func() {
		if b.Len() == 0 {
			b.WriteString("SELECT * FROM\n(\n")
		} else {
			b.WriteString("\n)\nUNION ALL\n(\n")
		}
	}

	for i := range srefs {

		if len(srefs[i].post) != 0 {

			next()

			if len(srefs[i].board) == 0 {
				// only postID
				q := `SELECT
	%d,
	xb.b_id,
	xb.b_name,
	xt.t_id,
	xt.t_name,
	xbp.p_name,
	'<' || xp.msgid || '>'
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
	xbp.g_p_id = xp.g_p_id
WHERE
	xbp.p_name LIKE '%s%%'
ORDER BY
	(xbp.b_id = %d AND xbp.t_id = %d) DESC,
	(xbp.b_id = %d) DESC,
	xbp.g_p_id DESC
LIMIT
	1`
				fmt.Fprintf(b, q, i+1, srefs[i].post, bid, tid, bid)

			} else {
				// board+postID
				q := `SELECT
	%d,
	xb.b_id,
	xb.b_name,
	xt.t_id,
	xt.t_name,
	xbp.p_name,
	'<' || xp.msgid || '>'
FROM
	ib0.bposts AS xbp
JOIN
	ib0.threads AS xt
ON
	xbp.b_id = xt.b_id AND xbp.t_id = xt.t_id
JOIN
	ib0.boards AS xb
ON
	xbp.b_id = xb.b_id
JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
WHERE
	xbp.p_name LIKE '%s%%' AND xb.b_name = '%s'
ORDER BY
	xbp.g_p_id DESC
LIMIT
	1`
				fmt.Fprintf(b, q, i+1, srefs[i].post, srefs[i].board)

			}
		} else if len(srefs[i].board) != 0 {
			// board
			// nothing, we don't need to look it up
		}
	}

	var rows *sql.Rows

	if b.Len() != 0 {
		b.WriteString("\n)") // finish it up

		q := b.String()

		sp.log.LogPrintf(DEBUG, "SQL for post references:\n%s", q)

		rows, err = sp.db.DB.Query(q)
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
		if len(srefs[i].post) != 0 {

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

			if len(srefs[i].board) == 0 {
				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(srefs[i].start),
						End:   uint(srefs[i].end),
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
				}
			} else {
				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(srefs[i].start),
						End:   uint(srefs[i].end),
					}
					r.Board = r_bname
					r.Thread = r_tname
					r.Post = r_pname

					refs = append(refs, r)
					inreplyto = append(inreplyto, r_msgid)
					sp.log.LogPrintf(DEBUG, "cref: %#v %q", r, r_msgid)
				}
			}
		} else {
			r := ib0.IBMessageReference{
				Start: uint(srefs[i].start),
				End:   uint(srefs[i].end),
			}

			r.Board = string(srefs[i].board)
			refs = append(refs, r)
			sp.log.LogPrintf(DEBUG, "bref: %#v", r)
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

func buildMsgIDArray(prefs []mm.FullMsgIDStr) string {
	var b strings.Builder

	b.WriteString("ARRAY[")
	for i := range prefs {
		if i != 0 {
			b.WriteByte(',')
		}
		b.WriteString(strings.Replace(string(prefs[i]), "'", "''", -1))
	}
	b.WriteByte(']')

	return b.String()
}

func (sp *PSQLIB) processReferencesOnIncoming(
	msg string, prefs []mm.FullMsgIDStr, bid boardID, tid postID) (
	refs []ib0.IBMessageReference, err error) {

	srefs := parseReferences(msg)

	if len(srefs) == 0 {
		return
	}

	qprefs := buildMsgIDArray(prefs)

	// build query
	b := &strings.Builder{}

	next := func() {
		if b.Len() == 0 {
			b.WriteString("SELECT * FROM\n(\n")
		} else {
			b.WriteString("\n)\nUNION ALL\n(\n")
		}
	}

	for i := range srefs {

		if len(srefs[i].post) != 0 {

			next()

			if len(srefs[i].board) == 0 {
				// only postID
				q := `SELECT
	%d,
	xb.b_id,
	xb.b_name,
	xt.t_id,
	xt.t_name,
	xbp.p_name
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
	xbp.g_p_id = xp.g_p_id
WHERE
	xbp.p_name LIKE '%s%%'
ORDER BY
	('<' || xp.msgid || '>' = ANY(%s)) DESC,
	(xbp.b_id = %d) DESC,
	xbp.g_p_id ASC
LIMIT
	1`
				fmt.Fprintf(b, q, i+1, srefs[i].post, qprefs, bid)

			} else {
				// board+postID
				q := `SELECT
	%d,
	xb.b_id,
	xb.b_name,
	xt.t_id,
	xt.t_name,
	xbp.p_name
FROM
	ib0.bposts AS xbp
JOIN
	ib0.threads AS xt
ON
	xbp.b_id = xt.b_id AND xbp.t_id = xt.t_id
JOIN
	ib0.boards AS xb
ON
	xbp.b_id = xb.b_id
JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
WHERE
	xbp.p_name LIKE '%s%%' AND xb.b_name = '%s'
ORDER BY
	('<' || xp.msgid || '>' = ANY(%s)) DESC,
	xbp.g_p_id ASC
LIMIT
	1`
				fmt.Fprintf(b, q, i+1, srefs[i].post, srefs[i].board, qprefs)

			}
		} else if len(srefs[i].board) != 0 {
			// board
			// nothing, we don't need to look it up
		}
	}

	var rows *sql.Rows

	if b.Len() != 0 {
		b.WriteString("\n)") // finish it up

		rows, err = sp.db.DB.Query(b.String())
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
		if len(srefs[i].post) != 0 {

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

			if len(srefs[i].board) == 0 {
				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(srefs[i].start),
						End:   uint(srefs[i].end),
					}
					r.Post = r_pname

					if r_bid != bid {
						r.Board = r_bname
						r.Thread = r_tname
					} else if r_tid != tid {
						r.Thread = r_tname
					}
					refs = append(refs, r)
				}
			} else {
				if r_id == i+1 {
					r := ib0.IBMessageReference{
						Start: uint(srefs[i].start),
						End:   uint(srefs[i].end),
					}
					r.Board = r_bname
					r.Thread = r_tname
					r.Post = r_pname

					refs = append(refs, r)
				}
			}
		} else {
			r := ib0.IBMessageReference{
				Start: uint(srefs[i].start),
				End:   uint(srefs[i].end),
			}

			r.Board = string(srefs[i].board)
			refs = append(refs, r)
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
