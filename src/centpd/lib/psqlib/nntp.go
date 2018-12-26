package psqlib

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/lib/pq"

	au "centpd/lib/asciiutils"
	. "centpd/lib/logx"
	"centpd/lib/mail"
	"centpd/lib/nntp"
)

var _ nntp.NNTPProvider = (*PSQLIB)(nil)

type groupState struct {
	bname string
	bid   boardID
	bpid  postID
	gpid  postID
}

type (
	Responder         = nntp.Responder
	AbstractResponder = nntp.AbstractResponder
	FullMsgID         = nntp.FullMsgID
	CoreMsgID         = nntp.CoreMsgID
	FullMsgIDStr      = nntp.FullMsgIDStr
	CoreMsgIDStr      = nntp.CoreMsgIDStr
	ConnState         = nntp.ConnState
)

func artnumInGroupsNum(cbid boardID, bids []boardID, bpids []postID) postID {
	for i, bid := range bids {
		if cbid == bid {
			return bpids[i]
		}
	}
	return 0
}

func artnumInGroups(cs *ConnState, bids []boardID, bpids []postID) postID {
	if cg, _ := cs.CurrentGroup.(*groupState); cg != nil {
		return artnumInGroupsNum(cg.bid, bids, bpids)
	}
	return 0
}

func getGroupState(cs *ConnState) *groupState {
	gs, _ := cs.CurrentGroup.(*groupState)
	return gs
}

func isGroupSelected(gs *groupState) bool {
	return gs != nil && gs.bid != 0
}

func currSelectedGroupID(cs *ConnState) boardID {
	if cg, _ := cs.CurrentGroup.(*groupState); cg != nil {
		return cg.bid
	}
	return 0
}

func (*PSQLIB) SupportsNewNews() bool     { return true }
func (*PSQLIB) SupportsOverByMsgID() bool { return true }
func (*PSQLIB) SupportsHdr() bool         { return true }

func (p *PSQLIB) SupportsIHave() bool  { return true }
func (p *PSQLIB) SupportsPost() bool   { return true }
func (p *PSQLIB) SupportsStream() bool { return true }

func unsafeCoreMsgIDToStr(b CoreMsgID) CoreMsgIDStr {
	return CoreMsgIDStr(unsafeBytesToStr(b))
}

var (
	errNotExist        = os.ErrNotExist
	errNoBoardSelected = errors.New("no board selected")
)

func (sp *PSQLIB) handleNNTPGetError(w Responder, nc nntpCopyer, e error) bool {
	if e == nil {
		// no error - handled successfuly
		return true
	}
	notexist := os.IsNotExist(e)
	if !notexist {
		sp.log.LogPrintf(ERROR, "handleNNTPGetError: err: %v", e)
	}
	if !nc.IsClosed() {
		// writer wasn't properly closed -- we should reset connection
		w.Abort()
		return true
	}
	// rest of errors are easier to handle
	if notexist {
		return false // this is pretty convenient
	} else if e == errNoBoardSelected {
		w.ResNoNewsgroupSelected()
	} else {
		w.ResInternalError(e)
	}
	return true
}

func (sp *PSQLIB) getArticleCommonByMsgID(
	nc nntpCopyer, w Responder, cs *ConnState, msgid CoreMsgID) bool {

	sid := unsafeCoreMsgIDToStr(msgid)
	e := sp.nntpObtainItemByMsgID(nc, cs, sid)
	return sp.handleNNTPGetError(w, nc, e)
}
func (sp *PSQLIB) GetArticleFullByMsgID(
	w Responder, cs *ConnState, msgid CoreMsgID) bool {

	nc := &fullNNTPCopyer{w: w}
	return sp.getArticleCommonByMsgID(nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleHeadByMsgID(
	w Responder, cs *ConnState, msgid CoreMsgID) bool {

	nc := &headNNTPCopyer{w: w}
	return sp.getArticleCommonByMsgID(nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleBodyByMsgID(
	w Responder, cs *ConnState, msgid CoreMsgID) bool {

	nc := &bodyNNTPCopyer{w: w}
	return sp.getArticleCommonByMsgID(nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleStatByMsgID(
	w Responder, cs *ConnState, msgid CoreMsgID) bool {

	sc := &statNNTPCopyer{w: w}
	return sp.getArticleCommonByMsgID(sc, w, cs, msgid)
}

func (sp *PSQLIB) getArticleCommonByNum(
	nc nntpCopyer, w Responder, cs *ConnState, num uint64) bool {

	e := sp.nntpObtainItemByNum(nc, cs, num)
	return sp.handleNNTPGetError(w, nc, e)
}
func (sp *PSQLIB) GetArticleFullByNum(
	w Responder, cs *ConnState, num uint64) bool {

	nc := &fullNNTPCopyer{w: w}
	return sp.getArticleCommonByNum(nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleHeadByNum(
	w Responder, cs *ConnState, num uint64) bool {

	nc := &headNNTPCopyer{w: w}
	return sp.getArticleCommonByNum(nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleBodyByNum(
	w Responder, cs *ConnState, num uint64) bool {

	nc := &bodyNNTPCopyer{w: w}
	return sp.getArticleCommonByNum(nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleStatByNum(
	w Responder, cs *ConnState, num uint64) bool {

	sc := &statNNTPCopyer{w: w}
	return sp.getArticleCommonByNum(sc, w, cs, num)
}

func (sp *PSQLIB) getArticleCommonByCurr(
	nc nntpCopyer, w Responder, cs *ConnState) bool {

	e := sp.nntpObtainItemByCurr(nc, cs)
	return sp.handleNNTPGetError(w, nc, e)
}
func (sp *PSQLIB) GetArticleFullByCurr(w Responder, cs *ConnState) bool {
	nc := &fullNNTPCopyer{w: w}
	return sp.getArticleCommonByCurr(nc, w, cs)
}
func (sp *PSQLIB) GetArticleHeadByCurr(w Responder, cs *ConnState) bool {
	nc := &headNNTPCopyer{w: w}
	return sp.getArticleCommonByCurr(nc, w, cs)
}
func (sp *PSQLIB) GetArticleBodyByCurr(w Responder, cs *ConnState) bool {
	nc := &bodyNNTPCopyer{w: w}
	return sp.getArticleCommonByCurr(nc, w, cs)
}
func (sp *PSQLIB) GetArticleStatByCurr(w Responder, cs *ConnState) bool {
	sc := &statNNTPCopyer{w: w}
	return sp.getArticleCommonByCurr(sc, w, cs)
}

func (sp *PSQLIB) SelectGroup(w Responder, cs *ConnState, group []byte) bool {
	sgroup := unsafeBytesToStr(group)

	var bid uint32
	var cnt uint64
	var lo, hi, g_lo sql.NullInt64

	q := st_list[st_NNTP_SelectGroup]

	err := sp.db.DB.
		QueryRow(q, sgroup).
		Scan(&bid, &cnt, &lo, &hi, &g_lo)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		w.ResInternalError(sp.sqlError("board-posts row query scan", err))
		return true
	}

	gs := getGroupState(cs)
	if gs == nil {
		gs = &groupState{}
		cs.CurrentGroup = gs
	}
	gs.bid = bid
	if gs.bname != sgroup {
		// sgroup is unsafe
		gs.bname = string(group)
	}
	gs.bpid = uint64(lo.Int64)
	gs.gpid = uint64(g_lo.Int64)

	if lo.Int64 > 0 {
		if hi.Int64 < lo.Int64 {
			hi = lo // paranoia
		}
		w.ResGroupSuccessfullySelected(
			cnt, uint64(lo.Int64), uint64(hi.Int64), sgroup)
	} else {
		w.ResGroupSuccessfullySelected(0, 0, 0, sgroup)
	}

	return true
}
func (sp *PSQLIB) SelectAndListGroup(
	w Responder, cs *ConnState, group []byte, rmin, rmax int64) bool {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		if len(group) == 0 {
			w.ResNoNewsgroupSelected()
			return true
		}
		if gs == nil {
			gs = &groupState{}
			cs.CurrentGroup = gs
		}
	}

	var sgroup string
	if len(group) != 0 {
		sgroup = unsafeBytesToStr(group)
	} else {
		sgroup = gs.bname
	}

	q := st_list[st_NNTP_SelectAndListGroup]

	rows, err := sp.db.DB.Query(q, sgroup, rmin, rmax)
	if err != nil {
		w.ResInternalError(sp.sqlError("board-posts query", err))
		return true
	}

	var dw io.WriteCloser
	for rows.Next() {
		var bid boardID
		var pcnt, lo, hi, g_lo, bpid sql.NullInt64
		err = rows.Scan(&bid, &pcnt, &lo, &hi, &g_lo, &bpid)
		if err != nil {
			rows.Close()
			err = sp.sqlError("board-post query rows scan", err)
			if dw == nil {
				w.ResInternalError(err)
			} else {
				w.Abort()
			}
			return true
		}
		if dw == nil {
			// we have something. do switch, send info about group
			gs.bid = bid
			if gs.bname != sgroup {
				// sgroup is unsafe
				gs.bname = string(group)
			}
			gs.bpid = postID(lo.Int64)
			gs.gpid = postID(g_lo.Int64)

			if lo.Int64 > 0 {
				if uint64(hi.Int64) < uint64(lo.Int64) {
					hi = lo // paranoia
				}
				w.ResArticleNumbersFollow(
					uint64(pcnt.Int64),
					uint64(lo.Int64), uint64(hi.Int64), sgroup)
			} else {
				w.ResArticleNumbersFollow(0, 0, 0, sgroup)
			}

			dw = w.DotWriter()
		}

		if bpid.Int64 != 0 {
			fmt.Fprintf(dw, "%d\n", bpid.Int64)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		err = sp.sqlError("board-post query rows iteration", err)
		if dw == nil {
			w.ResInternalError(err)
		} else {
			w.Abort()
		}
		return true
	}

	if dw != nil {
		dw.Close()
		return true
	} else {
		return false
	}
}
func (sp *PSQLIB) SelectNextArticle(w Responder, cs *ConnState) {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		w.ResNoNewsgroupSelected()
		return
	}
	if gs.bpid == 0 {
		w.ResCurrentArticleNumberIsInvalid()
		return
	}

	var nbpid postID
	var ngpid postID
	var msgid CoreMsgIDStr

	q := st_list[st_NNTP_SelectNextArticle]
	err := sp.db.DB.
		QueryRow(q, gs.bid, gs.bpid).
		Scan(&nbpid, &ngpid, &msgid)
	if err != nil {
		if err == sql.ErrNoRows {
			w.ResNoNextArticleInThisGroup()
			return
		}
		w.ResInternalError(sp.sqlError("posts row query scan", err))
		return
	}

	gs.bpid = nbpid
	gs.gpid = ngpid
	w.ResArticleFound(nbpid, msgid)
	return
}
func (sp *PSQLIB) SelectPrevArticle(w Responder, cs *ConnState) {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		w.ResNoNewsgroupSelected()
		return
	}
	if gs.bpid == 0 {
		w.ResCurrentArticleNumberIsInvalid()
		return
	}

	var nbpid postID
	var ngpid postID
	var msgid CoreMsgIDStr

	q := st_list[st_NNTP_SelectPrevArticle]
	err := sp.db.DB.
		QueryRow(q, gs.bid, gs.bpid).
		Scan(&nbpid, &ngpid, &msgid)
	if err != nil {
		if err == sql.ErrNoRows {
			w.ResNoPrevArticleInThisGroup()
			return
		}
		w.ResInternalError(sp.sqlError("posts row query scan", err))
		return
	}

	gs.bpid = nbpid
	gs.gpid = ngpid
	w.ResArticleFound(nbpid, msgid)
	return
}

func emptyWildmat(w []byte) bool {
	return len(w) == 0 || (len(w) == 1 && w[0] == '*')
}

func (sp *PSQLIB) ListNewNews(
	aw AbstractResponder, wildmat []byte, qt time.Time) {

	var rows *sql.Rows
	var err error
	var dw io.WriteCloser

	swildmat := unsafeBytesToStr(wildmat)
	wmany := swildmat == "*"
	wmgrp := !wmany && nntp.ValidGroupSlice(wildmat)

	if wmany || wmgrp {
		if wmany {
			q := st_list[st_NNTP_ListNewNews_all]
			rows, err = sp.db.DB.Query(q, qt)
		} else {
			q := st_list[st_NNTP_ListNewNews_one]
			rows, err = sp.db.DB.Query(q, qt, swildmat)
		}
		if err != nil {
			aw.GetResponder().ResInternalError(sp.sqlError("newnews query", err))
			return
		}

		dw = aw.OpenDotWriter()
		for rows.Next() {
			var msgid string

			err = rows.Scan(&msgid)
			if err != nil {
				rows.Close()
				sp.sqlError("newnews query rows scan", err)
				aw.Abort()
				return
			}

			fmt.Fprintf(dw, "<%s>\n", msgid)
		}
	} else {
		// TODO maybe we should use SQL LIKE to implement filtering?
		// that would be a little complicated, though
		wm := nntp.CompileWildmat(wildmat)

		q := st_list[st_NNTP_ListNewNews_all_group]
		rows, err = sp.db.DB.Query(q, qt)
		if err != nil {
			aw.GetResponder().ResInternalError(sp.sqlError("newnews query", err))
			return
		}

		dw = aw.OpenDotWriter()
		last_msgid := ""
		for rows.Next() {
			var msgid string
			var bname []byte

			err = rows.Scan(&msgid, &bname)
			if err != nil {
				rows.Close()
				sp.sqlError("newnews query rows scan", err)
				aw.Abort()
				return
			}

			if last_msgid == msgid {
				// skip duplicates
				continue
			}

			if wm.CheckBytes(bname) {
				last_msgid = msgid
				fmt.Fprintf(dw, "<%s>\n", msgid)
			}
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		sp.sqlError("newnews query rows iteration", err)
		aw.Abort()
		return
	}

	// TODO? maybe check error of close operation
	dw.Close()
}

func (sp *PSQLIB) ListNewGroups(aw AbstractResponder, qt time.Time) {
	// name hiwm lowm status
	// for now lets use status of "y"
	// TODO put something else in status when needed
	q := st_list[st_NNTP_ListNewGroups]
	rows, err := sp.db.DB.Query(q, qt)
	if err != nil {
		aw.GetResponder().ResInternalError(sp.sqlError("newgroups query", err))
		return
	}

	dw := aw.OpenDotWriter()
	for rows.Next() {
		var bname []byte
		var lo, hi sql.NullInt64

		err = rows.Scan(&bname, &lo, &hi)
		if err != nil {
			rows.Close()
			sp.sqlError("newgroups query rows scan", err)
			aw.Abort()
			return
		}

		if uint64(hi.Int64) < uint64(lo.Int64) {
			hi = lo // paranoia
		}

		fmt.Fprintf(dw, "%s %d %d y\n",
			bname, uint64(hi.Int64), uint64(lo.Int64))
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		sp.sqlError("newgroups query rows iteration", err)
		aw.Abort()
		return
	}

	dw.Close()
}

func (sp *PSQLIB) ListActiveGroups(aw AbstractResponder, wildmat []byte) {
	// name hiwm lowm status
	// for now lets use status of "y"
	// TODO put something else in status when needed

	var rows *sql.Rows
	var err error
	var wm nntp.Wildmat

	wmany := emptyWildmat(wildmat)
	wmgrp := !wmany && nntp.ValidGroupSlice(wildmat)

	if !wmany && !wmgrp {
		wm = nntp.CompileWildmat(wildmat)
	}

	if !wmgrp {
		q := st_list[st_NNTP_ListActiveGroups_all]
		rows, err = sp.db.DB.Query(q)
	} else {
		q := st_list[st_NNTP_ListActiveGroups_one]
		rows, err = sp.db.DB.Query(q, wildmat)
	}
	if err != nil {
		aw.GetResponder().ResInternalError(sp.sqlError("list active query", err))
		return
	}

	dw := aw.OpenDotWriter()
	for rows.Next() {
		var bname []byte
		var lo, hi sql.NullInt64

		err = rows.Scan(&bname, &lo, &hi)
		if err != nil {
			rows.Close()
			sp.sqlError("list active query rows scan", err)
			aw.Abort()
			return
		}

		if wm != nil && !wm.CheckBytes(bname) {
			continue
		}

		if uint64(hi.Int64) < uint64(lo.Int64) {
			hi = lo // paranoia
		}

		fmt.Fprintf(dw, "%s %d %d y\n",
			bname, uint64(hi.Int64), uint64(lo.Int64))
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		sp.sqlError("list active query rows iteration", err)
		aw.Abort()
		return
	}

	dw.Close()
}

func (sp *PSQLIB) ListNewsgroups(aw AbstractResponder, wildmat []byte) {
	// name\tdescription

	var rows *sql.Rows
	var err error
	var wm nntp.Wildmat

	wmany := emptyWildmat(wildmat)
	wmgrp := !wmany && nntp.ValidGroupSlice(wildmat)

	if !wmany && !wmgrp {
		wm = nntp.CompileWildmat(wildmat)
	}

	if !wmgrp {
		q := `SELECT b_name,bdesc FROM ib0.boards ORDER BY b_name`
		rows, err = sp.db.DB.Query(q)
	} else {
		q := `SELECT b_name,bdesc FROM ib0.boards WHERE b_name = $1 LIMIT 1`
		rows, err = sp.db.DB.Query(q, wildmat)
	}
	if err != nil {
		aw.GetResponder().
			ResInternalError(sp.sqlError("list newsgroups query", err))
		return
	}

	dw := aw.OpenDotWriter()
	for rows.Next() {
		var bname, bdesc string

		err = rows.Scan(&bname, &bdesc)
		if err != nil {
			rows.Close()
			sp.sqlError("list newsgroups query rows scan", err)
			aw.Abort()
			return
		}

		if wm != nil && !wm.CheckString(bname) {
			continue
		}

		bdesc = au.TrimWSString(bdesc)
		if bdesc == "" {
			// TODO should we do this? may be better for compatibility
			bdesc = "-"
		}

		fmt.Fprintf(dw, "%s\t%s\n", bname, bdesc)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		sp.sqlError("list active newsgroups rows iteration", err)
		aw.Abort()
		return
	}

	dw.Close()
}

var headerReplacer = strings.NewReplacer(
	"\t", " ",
	"\r", string(unicode.ReplacementChar),
	"\n", string(unicode.ReplacementChar),
	"\000", string(unicode.ReplacementChar))

// safeHeader prepares header value for OVER/HDR output
func safeHeader(s string) string {
	return headerReplacer.Replace(s)
}

func (sp *PSQLIB) printOver(
	w io.Writer, bpid postID, msgid CoreMsgIDStr,
	hsubject, hfrom, hdate, hrefs string,
	bnames []string, bpids []postID) {

	/*
		The first 8 fields MUST be the following, in order:
			"0" or article number (see below)
			Subject header content
			From header content
			Date header content
			Message-ID header content
			References header content
			:bytes metadata item
			:lines metadata item
		We also add Xref header field
	*/

	fmt.Fprintf(w,
		"%d\t%s\t%s\t%s\t<%s>\t%s\t%s\t%s\tXref: %s", bpid,
		safeHeader(hsubject), safeHeader(hfrom), safeHeader(hdate), msgid,
		safeHeader(hrefs), "", "", sp.instance)
	for i := range bnames {
		fmt.Fprintf(w, " %s:%d", bnames[i], bpids[i])
	}
	fmt.Fprintf(w, "\n")
}

// + ok: 224{ResOverviewInformationFollows}
// fail:
//   <ByMsgID>      430{ResNoArticleWithThatMsgID[false]}
//   <OverByRange>  412{ResNoNewsgroupSelected} 423{ResNoArticlesInThatRange[false]}
//   <XOverByRange> 412{ResNoNewsgroupSelected} 420{ResXNoArticles[false]}
//   <ByCurr>       412{ResNoNewsgroupSelected} 420{ResCurrentArticleNumberIsInvalid[false]}
func (sp *PSQLIB) GetOverByMsgID(
	w Responder, cs *ConnState, msgid CoreMsgID) bool {

	smsgid := unsafeCoreMsgIDToStr(msgid)

	var (
		bids   []boardID
		bpids  []postID
		bnames []string
		title  string

		hsubject, hfrom, hdate, hrefs sql.NullString
	)

	q := st_list[st_NNTP_GetOverByMsgID]
	err := sp.db.DB.
		QueryRow(q, smsgid).
		Scan(
			pq.Array(&bids),
			pq.Array(&bpids),
			pq.Array(&bnames),
			&title,
			&hsubject,
			&hfrom,
			&hdate,
			&hrefs)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		w.ResInternalError(sp.sqlError("overview query", err))
		return true
	}
	if !hsubject.Valid {
		hsubject.String = title
	}

	w.ResOverviewInformationFollows()
	dw := w.DotWriter()
	sp.printOver(dw, artnumInGroups(cs, bids, bpids), smsgid,
		hsubject.String, hfrom.String, hdate.String, hrefs.String,
		bnames, bpids)
	dw.Close()
	return true
}

func (sp *PSQLIB) GetOverByRange(
	w Responder, cs *ConnState, rmin, rmax int64) bool {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		w.ResNoNewsgroupSelected()
		return true
	}

	var dw io.WriteCloser

	q := st_list[st_NNTP_GetOverByRange]
	rows, err := sp.db.DB.Query(q, gs.bid, rmin, rmax)
	if err != nil {
		w.ResInternalError(sp.sqlError("overview query", err))
		return true
	}

	for rows.Next() {
		var (
			bids   []boardID
			bpids  []postID
			bnames []string
			cbpid  postID
			msgid  CoreMsgIDStr
			title  string

			hsubject, hfrom, hdate, hrefs sql.NullString
		)

		err = rows.Scan(
			pq.Array(&bids),
			pq.Array(&bpids),
			pq.Array(&bnames),
			&cbpid,
			&msgid,
			&title,
			&hsubject,
			&hfrom,
			&hdate,
			&hrefs)
		if err != nil {
			rows.Close()
			err = sp.sqlError("overview query rows scan", err)
			if dw == nil {
				w.ResInternalError(err)
			} else {
				w.Abort()
			}
			return true
		}
		if !hsubject.Valid {
			hsubject.String = title
		}

		if dw == nil {
			w.ResOverviewInformationFollows()
			dw = w.DotWriter()
		}

		sp.printOver(dw, cbpid, msgid,
			hsubject.String, hfrom.String, hdate.String, hrefs.String,
			bnames, bpids)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		err = sp.sqlError("overview query rows iteration", err)
		if dw == nil {
			w.ResInternalError(err)
		} else {
			w.Abort()
		}
		return true
	}

	if dw != nil {
		dw.Close()
		return true
	} else {
		return false
	}
}
func (sp *PSQLIB) GetXOverByRange(
	w Responder, cs *ConnState, rmin, rmax int64) bool {

	return sp.GetOverByRange(w, cs, rmin, rmax)
}
func (sp *PSQLIB) GetOverByCurr(w Responder, cs *ConnState) bool {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		w.ResNoNewsgroupSelected()
		return true
	}
	if gs.gpid == 0 {
		return false
	}

	var (
		bids   []boardID
		bpids  []postID
		bnames []string
		msgid  CoreMsgIDStr
		title  string

		hsubject, hfrom, hdate, hrefs sql.NullString
	)

	q := st_list[st_NNTP_GetOverByCurr]
	err := sp.db.DB.
		QueryRow(q, gs.gpid).
		Scan(
			pq.Array(&bids),
			pq.Array(&bpids),
			pq.Array(&bnames),
			&msgid,
			&title,
			&hsubject,
			&hfrom,
			&hdate,
			&hrefs)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		w.ResInternalError(sp.sqlError("overview query", err))
		return true
	}
	if !hsubject.Valid {
		hsubject.String = title
	}

	w.ResOverviewInformationFollows()
	dw := w.DotWriter()
	sp.printOver(dw, gs.bpid, msgid,
		hsubject.String, hfrom.String, hdate.String, hrefs.String,
		bnames, bpids)
	dw.Close()
	return true
}

func canonicalHeaderQueryStr(hdr []byte) (shdr string) {
	if len(hdr) == 0 || hdr[0] != ':' {
		shdr = mail.UnsafeCanonicalHeader(hdr)
	} else {
		nntp.ToLowerASCII(hdr)
		shdr = unsafeBytesToStr(hdr)
	}
	return
}

func bpidIfGroupEq(cbid, bid boardID, bpid postID) postID {
	if cbid == bid {
		return bpid
	}
	return 0
}

func (sp *PSQLIB) commonGetHdrByMsgID(
	w Responder, cs *ConnState, hdr []byte, msgid CoreMsgID, rfc bool) bool {

	sid := unsafeCoreMsgIDToStr(msgid)
	shdr := canonicalHeaderQueryStr(hdr)
	cbid := currSelectedGroupID(cs)

	var bid boardID
	var bpid postID
	var err error
	var h sql.NullString

	if shdr == "Message-ID" {

		q := st_list[st_NNTP_GetHdrByMsgID_msgid]

		err = sp.db.DB.
			QueryRow(q, msgid, cbid).
			Scan(&bid, &bpid)
		if err == nil {
			h.String = fmt.Sprintf("<%s>", sid)
		}

	} else if shdr == "Subject" {

		q := st_list[st_NNTP_GetHdrByMsgID_subject]

		var title string
		err = sp.db.DB.
			QueryRow(q, msgid, cbid).
			Scan(&bid, &bpid, &title, &h)
		if err == nil && !h.Valid {
			h.String = title
		}

	} else if shdr == "Bytes" || shdr == ":bytes" {
		// TODO
		w.PrintfLine("503 %q header unsupported", shdr)
		return true
	} else if shdr == "Lines" || shdr == ":lines" {
		// TODO
		w.PrintfLine("503 %q header unsupported", shdr)
		return true
	} else {

		q := st_list[st_NNTP_GetHdrByMsgID_any]

		err = sp.db.DB.
			QueryRow(q, msgid, cbid, shdr).
			Scan(&bid, &bpid, &h)

	}
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		w.ResInternalError(sp.sqlError("hdr query", err))
		return true
	}

	if rfc {
		w.ResHdrFollow()
		dw := w.DotWriter()
		fmt.Fprintf(dw, "%d %s\n",
			bpidIfGroupEq(cbid, bid, bpid), safeHeader(h.String))
		dw.Close()
	} else {
		w.ResXHdrFollow()
		dw := w.DotWriter()
		fmt.Fprintf(dw, "<%s> %s\n", sid, safeHeader(h.String))
		dw.Close()
	}

	return true
}
func (sp *PSQLIB) commonGetHdrByRange(
	w Responder, cs *ConnState, hdr []byte, rmin, rmax int64, rfc bool) bool {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		w.ResNoNewsgroupSelected()
		return true
	}

	shdr := canonicalHeaderQueryStr(hdr)

	var rowsscan = func(r *sql.Rows, pid *postID, h *sql.NullString) error {
		return r.Scan(pid, h)
	}

	var rows *sql.Rows
	var err error

	if shdr == "Message-ID" {

		q := st_list[st_NNTP_GetHdrByRange_msgid]

		rows, err = sp.db.DB.
			Query(q, gs.bid, rmin, rmax)

	} else if shdr == "Subject" {

		q := st_list[st_NNTP_GetHdrByRange_subject]

		rows, err = sp.db.DB.
			Query(q, gs.bid, rmin, rmax)

		rowsscan = func(r *sql.Rows, pid *postID, h *sql.NullString) error {
			var title string
			e := r.Scan(pid, &title, h)
			if e == nil && !h.Valid {
				h.String = title
			}
			return err
		}

	} else if shdr == "Bytes" || shdr == ":bytes" {
		// TODO
		w.PrintfLine("503 %q header unsupported", shdr)
		return true
	} else if shdr == "Lines" || shdr == ":lines" {
		// TODO
		w.PrintfLine("503 %q header unsupported", shdr)
		return true
	} else {

		q := st_list[st_NNTP_GetHdrByRange_any]

		rows, err = sp.db.DB.
			Query(q, gs.bid, rmin, rmax, shdr)

	}
	if err != nil {
		w.ResInternalError(sp.sqlError("hdr query", err))
		return true
	}

	var dw io.WriteCloser

	for rows.Next() {
		var pid postID
		var h sql.NullString

		err = rowsscan(rows, &pid, &h)
		if err != nil {
			rows.Close()
			err = sp.sqlError("hdr query rows scan", err)
			if dw == nil {
				w.ResInternalError(err)
			} else {
				w.Abort()
			}
			return true
		}

		if dw == nil {
			if rfc {
				w.ResHdrFollow()
			} else {
				w.ResXHdrFollow()
			}
			dw = w.DotWriter()
		}

		fmt.Fprintf(dw, "%d %s\n", pid, safeHeader(h.String))
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		err = sp.sqlError("hdr query rows iteration", err)
		if dw == nil {
			w.ResInternalError(err)
		} else {
			w.Abort()
		}
		return true
	}

	if dw != nil {
		dw.Close()
		return true
	} else {
		return false
	}
}
func (sp *PSQLIB) commonGetHdrByCurr(
	w Responder, cs *ConnState, hdr []byte, rfc bool) bool {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		w.ResNoNewsgroupSelected()
		return true
	}
	if gs.gpid == 0 {
		return false
	}

	shdr := canonicalHeaderQueryStr(hdr)

	var rowscan = func(r *sql.Row, h *sql.NullString) error {
		return r.Scan(h)
	}

	var err error
	var h sql.NullString
	var row *sql.Row

	if shdr == "Message-ID" {

		q := st_list[st_NNTP_GetHdrByCurr_msgid]

		row = sp.db.DB.QueryRow(q, gs.gpid)

	} else if shdr == "Subject" {

		q := st_list[st_NNTP_GetHdrByCurr_subject]

		row = sp.db.DB.QueryRow(q, gs.gpid)

		rowscan = func(r *sql.Row, h *sql.NullString) error {
			var title string
			e := r.Scan(&title, h)
			if e == nil && !h.Valid {
				h.String = title
			}
			return e
		}

	} else if shdr == "Bytes" || shdr == ":bytes" {
		// TODO
		w.PrintfLine("503 %q header unsupported", shdr)
		return true
	} else if shdr == "Lines" || shdr == ":lines" {
		// TODO
		w.PrintfLine("503 %q header unsupported", shdr)
		return true
	} else {

		q := st_list[st_NNTP_GetHdrByCurr_any]

		row = sp.db.DB.QueryRow(q, gs.gpid, shdr)

	}
	err = rowscan(row, &h)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		w.ResInternalError(sp.sqlError("hdr query", err))
		return true
	}

	if rfc {
		w.ResHdrFollow()
	} else {
		w.ResXHdrFollow()
	}

	dw := w.DotWriter()
	fmt.Fprintf(dw, "%d %s\n", gs.bpid, safeHeader(h.String))
	dw.Close()

	return true
}
func (sp *PSQLIB) GetHdrByMsgID(
	w Responder, cs *ConnState, hdr []byte, msgid CoreMsgID) bool {

	return sp.commonGetHdrByMsgID(w, cs, hdr, msgid, true)
}
func (sp *PSQLIB) GetHdrByRange(
	w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool {

	return sp.commonGetHdrByRange(w, cs, hdr, rmin, rmax, true)
}
func (sp *PSQLIB) GetHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool {
	return sp.commonGetHdrByCurr(w, cs, hdr, true)
}
func (sp *PSQLIB) GetXHdrByMsgID(
	w Responder, hdr []byte, msgid CoreMsgID) bool {

	return sp.commonGetHdrByMsgID(w, nil, hdr, msgid, false)
}
func (sp *PSQLIB) GetXHdrByRange(
	w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool {

	return sp.commonGetHdrByRange(w, cs, hdr, rmin, rmax, false)
}
func (sp *PSQLIB) GetXHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool {
	return sp.commonGetHdrByCurr(w, cs, hdr, false)
}
