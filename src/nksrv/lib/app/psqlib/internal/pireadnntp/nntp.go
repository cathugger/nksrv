package pireadnntp

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

	"nksrv/lib/app/psqlib/internal/pibase"
	"nksrv/lib/mail"
	"nksrv/lib/nntp"
	. "nksrv/lib/utils/logx"
	au "nksrv/lib/utils/text/asciiutils"
)

type (
	boardID = pibase.TBoardID
	postID  = pibase.TPostID
)

func nntpAbortOnErr(err error) {
	if err != nil {
		// TODO wrapping
		panic(nntp.ErrAbortHandler)
	}
}

type groupState struct {
	bname string
	bid   boardID
	bpid  postID
	gpid  postID
}

type (
	Responder         = nntp.Responder
	AbstractResponder = nntp.AbstractResponder
	TFullMsgID        = nntp.TFullMsgID
	TCoreMsgID        = nntp.TCoreMsgID
	TFullMsgIDStr     = nntp.TFullMsgIDStr
	TCoreMsgIDStr     = nntp.TCoreMsgIDStr
	ConnState         = nntp.ConnState
)

func artnumInGroups(cs *ConnState, bids []int64, bpids []int64) postID {
	if cg, _ := cs.CurrentGroup.(*groupState); cg != nil {
		for i, bid := range bids {
			if cg.bid == boardID(bid) {
				return postID(bpids[i])
			}
		}
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

func unsafeCoreMsgIDToStr(b TCoreMsgID) TCoreMsgIDStr {
	return TCoreMsgIDStr(unsafeBytesToStr(b))
}

var (
	errNotExist        = os.ErrNotExist
	errNoBoardSelected = errors.New("no board selected")
)

func handleNNTPGetError(sp *pibase.PSQLIB, w Responder, nc nntpCopyer, e error) bool {
	if e == nil {
		// no error - handled successfuly
		return true
	}
	notexist := os.IsNotExist(e)
	if !notexist {
		sp.Log.LogPrintf(ERROR, "handleNNTPGetError: err: %v", e)
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
		nntpAbortOnErr(w.ResNoNewsgroupSelected())
	} else {
		nntpAbortOnErr(w.ResInternalError(e))
	}
	return true
}

func GetArticleCommonByMsgID(
	sp *pibase.PSQLIB, nc nntpCopyer, w Responder, cs *ConnState,
	msgid TCoreMsgID) bool {

	sid := unsafeCoreMsgIDToStr(msgid)
	e := nntpObtainItemByMsgID(sp, nc, cs, sid)
	return handleNNTPGetError(sp, w, nc, e)
}

func GetArticleCommonByNum(
	sp *pibase.PSQLIB, nc nntpCopyer, w Responder, cs *ConnState,
	num uint64) bool {

	e := nntpObtainItemByNum(sp, nc, cs, num)
	return handleNNTPGetError(sp, w, nc, e)
}

func GetArticleCommonByCurr(
	sp *pibase.PSQLIB, nc nntpCopyer, w Responder, cs *ConnState) bool {

	e := nntpObtainItemByCurr(sp, nc, cs)
	return handleNNTPGetError(sp, w, nc, e)
}

func SelectGroup(
	sp *pibase.PSQLIB, w Responder, cs *ConnState, group []byte) bool {

	sgroup := unsafeBytesToStr(group)

	var bid uint32
	var cnt uint64
	var lo, hi, g_lo sql.NullInt64

	err := sp.StPrep[pibase.St_nntp_select].
		QueryRow(sgroup).
		Scan(&bid, &cnt, &lo, &hi, &g_lo)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		nntpAbortOnErr(w.ResInternalError(
			sp.SQLError("board-posts row query scan", err)))
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
		nntpAbortOnErr(w.ResGroupSuccessfullySelected(
			cnt, uint64(lo.Int64), uint64(hi.Int64), sgroup))
	} else {
		nntpAbortOnErr(w.ResGroupSuccessfullySelected(0, 0, 0, sgroup))
	}

	return true
}
func SelectAndListGroup(
	sp *pibase.PSQLIB, w Responder, cs *ConnState,
	group []byte, rmin, rmax int64) bool {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		if len(group) == 0 {
			nntpAbortOnErr(w.ResNoNewsgroupSelected())
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

	rows, err := sp.StPrep[pibase.St_nntp_select_and_list].
		Query(sgroup, rmin, rmax)
	if err != nil {
		nntpAbortOnErr(w.ResInternalError(
			sp.SQLError("board-posts query", err)))
		return true
	}

	var dw io.WriteCloser
	for rows.Next() {
		var bid boardID
		var pcnt, lo, hi, g_lo, bpid sql.NullInt64
		err = rows.Scan(&bid, &pcnt, &lo, &hi, &g_lo, &bpid)
		if err != nil {
			rows.Close()
			err = sp.SQLError("board-post query rows scan", err)
			if dw == nil {
				nntpAbortOnErr(w.ResInternalError(err))
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
				nntpAbortOnErr(w.ResArticleNumbersFollow(
					uint64(pcnt.Int64),
					uint64(lo.Int64), uint64(hi.Int64), sgroup))
			} else {
				nntpAbortOnErr(w.ResArticleNumbersFollow(0, 0, 0, sgroup))
			}

			dw = w.DotWriter()
		}

		if bpid.Int64 != 0 {
			fmt.Fprintf(dw, "%d\n", bpid.Int64)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		err = sp.SQLError("board-post query rows iteration", err)
		if dw == nil {
			nntpAbortOnErr(w.ResInternalError(err))
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
func SelectNextArticle(sp *pibase.PSQLIB, w Responder, cs *ConnState) {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		nntpAbortOnErr(w.ResNoNewsgroupSelected())
		return
	}
	if gs.bpid == 0 {
		nntpAbortOnErr(w.ResCurrentArticleNumberIsInvalid())
		return
	}

	var nbpid postID
	var ngpid postID
	var msgid TCoreMsgIDStr

	err := sp.StPrep[pibase.St_nntp_next].
		QueryRow(gs.bid, gs.bpid).
		Scan(&nbpid, &ngpid, &msgid)
	if err != nil {
		if err == sql.ErrNoRows {
			nntpAbortOnErr(w.ResNoNextArticleInThisGroup())
			return
		}
		nntpAbortOnErr(w.ResInternalError(
			sp.SQLError("posts row query scan", err)))
		return
	}

	gs.bpid = nbpid
	gs.gpid = ngpid
	nntpAbortOnErr(w.ResArticleFound(nbpid, msgid))
}
func SelectPrevArticle(sp *pibase.PSQLIB, w Responder, cs *ConnState) {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		nntpAbortOnErr(w.ResNoNewsgroupSelected())
		return
	}
	if gs.bpid == 0 {
		nntpAbortOnErr(w.ResCurrentArticleNumberIsInvalid())
		return
	}

	var nbpid postID
	var ngpid postID
	var msgid TCoreMsgIDStr

	err := sp.StPrep[pibase.St_nntp_last].
		QueryRow(gs.bid, gs.bpid).
		Scan(&nbpid, &ngpid, &msgid)
	if err != nil {
		if err == sql.ErrNoRows {
			nntpAbortOnErr(w.ResNoPrevArticleInThisGroup())
			return
		}
		nntpAbortOnErr(w.ResInternalError(
			sp.SQLError("posts row query scan", err)))
		return
	}

	gs.bpid = nbpid
	gs.gpid = ngpid
	nntpAbortOnErr(w.ResArticleFound(nbpid, msgid))
}

func emptyWildmat(w []byte) bool {
	return len(w) == 0 || (len(w) == 1 && w[0] == '*')
}

func ListNewNews(
	sp *pibase.PSQLIB, aw AbstractResponder, wildmat []byte, qt time.Time) {

	var rows *sql.Rows
	var err error
	var dw io.WriteCloser

	swildmat := unsafeBytesToStr(wildmat)
	wmany := swildmat == "*"
	wmgrp := !wmany && nntp.ValidGroupSlice(wildmat)

	if wmany || wmgrp {
		if wmany {
			rows, err = sp.StPrep[pibase.St_nntp_newnews_all].Query(qt)
		} else {
			rows, err = sp.StPrep[pibase.St_nntp_newnews_one].Query(qt, swildmat)
		}
		if err != nil {
			nntpAbortOnErr(aw.GetResponder().ResInternalError(
				sp.SQLError("newnews query", err)))
			return
		}

		dw, err = aw.OpenDotWriter()
		if err != nil {
			rows.Close()
			nntpAbortOnErr(err)
		}
		for rows.Next() {
			var msgid string

			err = rows.Scan(&msgid)
			if err != nil {
				rows.Close()
				_ = sp.SQLError("newnews query rows scan", err)
				aw.Abort()
				return
			}

			fmt.Fprintf(dw, "<%s>\n", msgid)
		}
	} else {
		// TODO maybe we should use SQL LIKE to implement filtering?
		// that would be a little bit complicated, though
		wm := nntp.CompileWildmat(wildmat)

		rows, err = sp.StPrep[pibase.St_nntp_newnews_all_group].Query(qt)
		if err != nil {
			nntpAbortOnErr(aw.GetResponder().ResInternalError(
				sp.SQLError("newnews query", err)))
			return
		}

		dw, err = aw.OpenDotWriter()
		if err != nil {
			rows.Close()
			nntpAbortOnErr(err)
		}
		last_msgid := ""
		for rows.Next() {
			var msgid string
			var bname []byte

			err = rows.Scan(&msgid, &bname)
			if err != nil {
				rows.Close()
				_ = sp.SQLError("newnews query rows scan", err)
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
		_ = sp.SQLError("newnews query rows iteration", err)
		aw.Abort()
		return
	}

	nntpAbortOnErr(dw.Close())
}

func ListNewGroups(sp *pibase.PSQLIB, aw AbstractResponder, qt time.Time) {
	// name hiwm lowm status
	// for now lets use status of "y"
	// TODO put something else in status when needed
	rows, err := sp.StPrep[pibase.St_nntp_newgroups].Query(qt)
	if err != nil {
		nntpAbortOnErr(aw.GetResponder().ResInternalError(
			sp.SQLError("newgroups query", err)))
		return
	}
	defer func() {
		if err != nil {
			rows.Close()
		}
	}()

	dw, err := aw.OpenDotWriter()
	nntpAbortOnErr(err)

	for rows.Next() {
		var bname []byte
		var lo, hi sql.NullInt64

		err = rows.Scan(&bname, &lo, &hi)
		if err != nil {
			_ = sp.SQLError("newgroups query rows scan", err)
			aw.Abort()
			return
		}

		if uint64(hi.Int64) < uint64(lo.Int64) {
			hi = lo // paranoia
		}

		_, err = fmt.Fprintf(dw, "%s %d %d y\n",
			bname, uint64(hi.Int64), uint64(lo.Int64))
		nntpAbortOnErr(err)
	}
	if err = rows.Err(); err != nil {
		_ = sp.SQLError("newgroups query rows iteration", err)
		aw.Abort()
		return
	}

	nntpAbortOnErr(dw.Close())
}

func ListActiveGroups(sp *pibase.PSQLIB, aw AbstractResponder, wildmat []byte) {
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
		rows, err = sp.StPrep[pibase.St_nntp_listactive_all].Query()
	} else {
		rows, err = sp.StPrep[pibase.St_nntp_listactive_one].Query(wildmat)
	}
	if err != nil {
		nntpAbortOnErr(aw.GetResponder().ResInternalError(
			sp.SQLError("list active query", err)))
		return
	}

	defer func() {
		if err != nil {
			rows.Close()
		}
	}()

	dw, err := aw.OpenDotWriter()
	nntpAbortOnErr(err)

	for rows.Next() {
		var bname []byte
		var lo, hi sql.NullInt64

		err = rows.Scan(&bname, &lo, &hi)
		if err != nil {
			_ = sp.SQLError("list active query rows scan", err)
			aw.Abort()
			return
		}

		if wm != nil && !wm.CheckBytes(bname) {
			continue
		}

		if uint64(hi.Int64) < uint64(lo.Int64) {
			hi = lo // paranoia
		}

		_, err = fmt.Fprintf(dw, "%s %d %d y\n",
			bname, uint64(hi.Int64), uint64(lo.Int64))
		nntpAbortOnErr(err)
	}
	if err = rows.Err(); err != nil {
		_ = sp.SQLError("list active query rows iteration", err)
		aw.Abort()
		return
	}

	nntpAbortOnErr(dw.Close())
}

func ListNewsgroups(sp *pibase.PSQLIB, aw AbstractResponder, wildmat []byte) {
	// name[tab]description

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
		rows, err = sp.DB.DB.Query(q)
	} else {
		q := `SELECT b_name,bdesc FROM ib0.boards WHERE b_name = $1 LIMIT 1`
		rows, err = sp.DB.DB.Query(q, wildmat)
	}
	if err != nil {
		nntpAbortOnErr(aw.GetResponder().ResInternalError(
			sp.SQLError("list newsgroups query", err)))
		return
	}

	defer func() {
		if err != nil {
			rows.Close()
		}
	}()

	dw, err := aw.OpenDotWriter()
	nntpAbortOnErr(err)

	for rows.Next() {
		var bname, bdesc string

		err = rows.Scan(&bname, &bdesc)
		if err != nil {
			_ = sp.SQLError("list newsgroups query rows scan", err)
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

		_, err = fmt.Fprintf(dw, "%s\t%s\n", bname, bdesc)
		nntpAbortOnErr(err)
	}
	if err = rows.Err(); err != nil {
		_ = sp.SQLError("list active newsgroups rows iteration", err)
		aw.Abort()
		return
	}

	nntpAbortOnErr(dw.Close())
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

func printOver(
	sp *pibase.PSQLIB, w io.Writer, bpid postID, msgid TCoreMsgIDStr,
	hsubject, hfrom, hdate, hrefs string,
	bnames []string, bpids []int64) {

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

	// some newsreaders (looking at you Pan) misbehave without Subject
	if hsubject == "" {
		hsubject = "(No Subject)"
	}

	fmt.Fprintf(w,
		"%d\t%s\t%s\t%s\t<%s>\t%s\t%s\t%s\tXref: %s", bpid,
		safeHeader(hsubject), safeHeader(hfrom), safeHeader(hdate), msgid,
		safeHeader(hrefs), "", "", sp.Instance)
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
func GetOverByMsgID(
	sp *pibase.PSQLIB, w Responder, cs *ConnState, msgid TCoreMsgID) bool {

	smsgid := unsafeCoreMsgIDToStr(msgid)

	var (
		bids   []int64
		bpids  []int64
		bnames []string
		title  string

		hsubject, hfrom, hdate, hrefs sql.NullString

		isbanned bool
	)

	err := sp.StPrep[pibase.St_nntp_over_msgid].
		QueryRow(smsgid).
		Scan(
			pq.Array(&bids),
			pq.Array(&bpids),
			pq.Array(&bnames),
			&title,
			&hsubject,
			&hfrom,
			&hdate,
			&hrefs,
			&isbanned)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		nntpAbortOnErr(w.ResInternalError(sp.SQLError("overview query", err)))
		return true
	}
	if isbanned {
		// this kind of signaling so far
		return false
	}
	if !hsubject.Valid {
		hsubject.String = title
	}

	nntpAbortOnErr(w.ResOverviewInformationFollows())
	dw := w.DotWriter()
	printOver(sp, dw, artnumInGroups(cs, bids, bpids), smsgid,
		hsubject.String, hfrom.String, hdate.String, hrefs.String,
		bnames, bpids)
	dw.Close()
	return true
}

func GetOverByRange(
	sp *pibase.PSQLIB, w Responder, cs *ConnState, rmin, rmax int64) bool {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		nntpAbortOnErr(w.ResNoNewsgroupSelected())
		return true
	}

	var dw io.WriteCloser

	rows, err := sp.StPrep[pibase.St_nntp_over_range].
		Query(gs.bid, rmin, rmax)
	if err != nil {
		nntpAbortOnErr(w.ResInternalError(sp.SQLError("overview query", err)))
		return true
	}

	for rows.Next() {
		var (
			bids   []int64
			bpids  []int64
			bnames []string
			cbpid  postID
			msgid  TCoreMsgIDStr
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
			err = sp.SQLError("overview query rows scan", err)
			if dw == nil {
				nntpAbortOnErr(w.ResInternalError(err))
			} else {
				w.Abort()
			}
			return true
		}
		if !hsubject.Valid {
			hsubject.String = title
		}

		if dw == nil {
			nntpAbortOnErr(w.ResOverviewInformationFollows())
			dw = w.DotWriter()
		}

		printOver(sp, dw, cbpid, msgid,
			hsubject.String, hfrom.String, hdate.String, hrefs.String,
			bnames, bpids)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		err = sp.SQLError("overview query rows iteration", err)
		if dw == nil {
			nntpAbortOnErr(w.ResInternalError(err))
		} else {
			w.Abort()
		}
		return true
	}

	if dw != nil {
		nntpAbortOnErr(dw.Close())
		return true
	} else {
		return false
	}
}

func GetOverByCurr(sp *pibase.PSQLIB, w Responder, cs *ConnState) bool {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		nntpAbortOnErr(w.ResNoNewsgroupSelected())
		return true
	}
	if gs.gpid == 0 {
		return false
	}

	var (
		bids   []int64
		bpids  []int64
		bnames []string
		msgid  TCoreMsgIDStr
		title  string

		hsubject, hfrom, hdate, hrefs sql.NullString
	)

	err := sp.StPrep[pibase.St_nntp_over_curr].
		QueryRow(gs.gpid).
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
		nntpAbortOnErr(w.ResInternalError(sp.SQLError("overview query", err)))
		return true
	}
	if !hsubject.Valid {
		hsubject.String = title
	}

	nntpAbortOnErr(w.ResOverviewInformationFollows())
	dw := w.DotWriter()
	printOver(sp, dw, gs.bpid, msgid,
		hsubject.String, hfrom.String, hdate.String, hrefs.String,
		bnames, bpids)
	nntpAbortOnErr(dw.Close())
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

func CommonGetHdrByMsgID(
	sp *pibase.PSQLIB,
	w Responder, cs *ConnState, hdr []byte, msgid TCoreMsgID, rfc bool) bool {

	sid := unsafeCoreMsgIDToStr(msgid)
	shdr := canonicalHeaderQueryStr(hdr)
	cbid := currSelectedGroupID(cs)

	var bid boardID
	var bpid postID
	var err error
	var h sql.NullString
	isbanned := false

	if shdr == "Message-ID" {

		err = sp.StPrep[pibase.St_nntp_hdr_msgid_msgid].
			QueryRow(msgid, cbid).
			Scan(&bid, &bpid, &isbanned)
		if err == nil {
			h.String = fmt.Sprintf("<%s>", sid)
		}

	} else if shdr == "Subject" {

		var title string

		err = sp.StPrep[pibase.St_nntp_hdr_msgid_subject].
			QueryRow(msgid, cbid).
			Scan(&bid, &bpid, &title, &h, &isbanned)
		if err == nil && !h.Valid {
			h.String = title
		}

	} else if shdr == "Bytes" || shdr == ":bytes" {
		// TODO
		nntpAbortOnErr(w.PrintfLine("503 %q header unsupported", shdr))
		return true
	} else if shdr == "Lines" || shdr == ":lines" {
		// TODO
		nntpAbortOnErr(w.PrintfLine("503 %q header unsupported", shdr))
		return true
	} else {

		err = sp.StPrep[pibase.St_nntp_hdr_msgid_any].
			QueryRow(msgid, cbid, shdr).
			Scan(&bid, &bpid, &h, &isbanned)

	}
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		nntpAbortOnErr(w.ResInternalError(sp.SQLError("hdr query", err)))
		return true
	}
	if isbanned {
		// this kind of signaling so far
		return false
	}

	if rfc {
		nntpAbortOnErr(w.ResHdrFollow())
		dw := w.DotWriter()
		fmt.Fprintf(dw, "%d %s\n",
			bpidIfGroupEq(cbid, bid, bpid), safeHeader(h.String))
		nntpAbortOnErr(dw.Close())
	} else {
		nntpAbortOnErr(w.ResXHdrFollow())
		dw := w.DotWriter()
		fmt.Fprintf(dw, "<%s> %s\n", sid, safeHeader(h.String))
		nntpAbortOnErr(dw.Close())
	}

	return true
}
func CommonGetHdrByRange(
	sp *pibase.PSQLIB,
	w Responder, cs *ConnState, hdr []byte, rmin, rmax int64, rfc bool) bool {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		nntpAbortOnErr(w.ResNoNewsgroupSelected())
		return true
	}

	shdr := canonicalHeaderQueryStr(hdr)

	var rowsscan = func(r *sql.Rows, pid *postID, h *sql.NullString) error {
		return r.Scan(pid, h)
	}

	var rows *sql.Rows
	var err error

	if shdr == "Message-ID" {

		rows, err = sp.StPrep[pibase.St_nntp_hdr_range_msgid].
			Query(gs.bid, rmin, rmax)

	} else if shdr == "Subject" {

		rows, err = sp.StPrep[pibase.St_nntp_hdr_range_subject].
			Query(gs.bid, rmin, rmax)

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
		nntpAbortOnErr(w.PrintfLine("503 %q header unsupported", shdr))
		return true
	} else if shdr == "Lines" || shdr == ":lines" {
		// TODO
		nntpAbortOnErr(w.PrintfLine("503 %q header unsupported", shdr))
		return true
	} else {

		rows, err = sp.StPrep[pibase.St_nntp_hdr_range_any].
			Query(gs.bid, rmin, rmax, shdr)

	}
	if err != nil {
		nntpAbortOnErr(w.ResInternalError(sp.SQLError("hdr query", err)))
		return true
	}

	var dw io.WriteCloser

	for rows.Next() {
		var pid postID
		var h sql.NullString

		err = rowsscan(rows, &pid, &h)
		if err != nil {
			rows.Close()
			err = sp.SQLError("hdr query rows scan", err)
			if dw == nil {
				nntpAbortOnErr(w.ResInternalError(err))
			} else {
				w.Abort()
			}
			return true
		}

		if dw == nil {
			if rfc {
				nntpAbortOnErr(w.ResHdrFollow())
			} else {
				nntpAbortOnErr(w.ResXHdrFollow())
			}
			dw = w.DotWriter()
		}

		fmt.Fprintf(dw, "%d %s\n", pid, safeHeader(h.String))
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		err = sp.SQLError("hdr query rows iteration", err)
		if dw == nil {
			nntpAbortOnErr(w.ResInternalError(err))
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
func CommonGetHdrByCurr(
	sp *pibase.PSQLIB,
	w Responder, cs *ConnState, hdr []byte, rfc bool) bool {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		nntpAbortOnErr(w.ResNoNewsgroupSelected())
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

		row = sp.StPrep[pibase.St_nntp_hdr_curr_msgid].QueryRow(gs.gpid)

	} else if shdr == "Subject" {

		row = sp.StPrep[pibase.St_nntp_hdr_curr_subject].QueryRow(gs.gpid)

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
		nntpAbortOnErr(w.PrintfLine("503 %q header unsupported", shdr))
		return true
	} else if shdr == "Lines" || shdr == ":lines" {
		// TODO
		nntpAbortOnErr(w.PrintfLine("503 %q header unsupported", shdr))
		return true
	} else {

		row = sp.StPrep[pibase.St_nntp_hdr_curr_any].QueryRow(gs.gpid, shdr)

	}
	err = rowscan(row, &h)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		nntpAbortOnErr(w.ResInternalError(sp.SQLError("hdr query", err)))
		return true
	}

	if rfc {
		nntpAbortOnErr(w.ResHdrFollow())
	} else {
		nntpAbortOnErr(w.ResXHdrFollow())
	}

	dw := w.DotWriter()
	fmt.Fprintf(dw, "%d %s\n", gs.bpid, safeHeader(h.String))
	nntpAbortOnErr(dw.Close())

	return true
}
