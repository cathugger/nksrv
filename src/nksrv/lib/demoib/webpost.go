package demoib

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"nksrv/lib/date"
	"nksrv/lib/mail/form"
	"nksrv/lib/mailib"
	mm "nksrv/lib/minimail"
	tu "nksrv/lib/textutils"
	ib0 "nksrv/lib/webib0"
)

var FileFields = ib0.IBWebFormFileFields

type formFileOpener struct {
}

var _ form.FileOpener = formFileOpener{}

func (formFileOpener) OpenFile() (*os.File, error) {
	return ioutil.TempFile("", "webpost-")
}

var textFields = []string{
	ib0.IBWebFormTextTitle,
	ib0.IBWebFormTextName,
	ib0.IBWebFormTextMessage,
	ib0.IBWebFormTextOptions,
}

// FIXME: this probably in future should go thru some sort of abstractation

func (IBProviderDemo) IBGetPostParams() (
	*form.ParserParams, form.FileOpener, []string) {

	return &form.DefaultParserParams, formFileOpener{}, textFields
}

type CoreMsgIDStr = mm.CoreMsgIDStr

type postedInfo = ib0.IBPostedInfo

var lineReplacer = strings.NewReplacer(
	"\r", "",
	"\n", " ",
	"\t", " ")

func optimiseFormLine(line string) (s string) {
	s = lineReplacer.Replace(line)
	s = norm.NFC.String(s)
	return
}

func commonNewPost(
	w http.ResponseWriter, r *http.Request, f form.Form, board, thread string, isReply bool) (
	rInfo postedInfo, err error, _ int) {

	defer func() {
		if err != nil {
			f.RemoveAll()
		}
	}()

	fntitle := ib0.IBWebFormTextTitle
	fnmessage := ib0.IBWebFormTextMessage

	// XXX more fields
	if len(f.Values[fntitle]) != 1 ||
		len(f.Values[fnmessage]) != 1 {

		return rInfo, errInvalidSubmission, http.StatusBadRequest
	}

	xftitle := f.Values[fntitle][0]
	xfmessage := f.Values[fnmessage][0]

	fmt.Fprintf(os.Stderr, "form fields: xftitle(%q) xfmessage(%q)\n",
		xftitle, xfmessage)

	if !utf8.ValidString(xftitle) ||
		!utf8.ValidString(xfmessage) {

		return rInfo, errBadSubmissionEncoding, http.StatusBadRequest
	}

	// get info about board, its limits and shit. does it even exists?
	if !isReply {
		if board != "test" {
			return rInfo, errNoSuchBoard, http.StatusNotFound
		}
		rInfo.Board = board
	} else {
		if board != "test" {
			return rInfo, errNoSuchBoard, http.StatusNotFound
		}
		rInfo.Board = board

		if len(thread) < 4 || thread[:4] != "0123" {
			return rInfo, errNoSuchThread, http.StatusNotFound
		}
		rInfo.ThreadID = thread
	}

	// use normalised forms
	// theorically, normalisation could increase size sometimes, which could lead to rejection of previously-fitting message
	// but it's better than accepting too big message, as that could lead to bad things later on
	pTitle := strings.TrimSpace(optimiseFormLine(xftitle))
	pMessage := tu.NormalizeTextMessage(xfmessage)
	fmt.Fprintf(os.Stderr,
		"form fields after processing: Title{%q} Message{%q}\n",
		pTitle, pMessage)

	// at this point message should be checked
	// we should calculate proper file names here
	// should we move files before or after writing to database?
	// maybe we should update database in 2 stages, first before, and then after?
	// or maybe we should keep journal to ensure consistency after crash?
	// decision: first write to database, then to file system. on crash, scan files table and check if files are in place (by fid).
	// there still can be the case where there are left untracked files in file system. they could be manually scanned, and damage is low.

	// process files
	f.RemoveAll()

	// postprocess
	tu := date.NowTimeUnix()
	// lets think of post ID there
	pMessageID := mailib.NewRandomMessageID(tu, "test.invalid")
	pID := mailib.HashPostID_SHA1(pMessageID)

	if !isReply {
		rInfo.ThreadID = pID
	}
	rInfo.PostID = pID
	rInfo.MessageID = mm.CutMessageIDStr(pMessageID)
	return
}

func (IBProviderDemo) IBDefaultBoardInfo() ib0.IBNewBoardInfo {
	return ib0.IBNewBoardInfo{
		Name:           "",
		Description:    "",
		ThreadsPerPage: 10,
		MaxActivePages: 10,
		MaxPages:       15,
	}
}

func (IBProviderDemo) IBPostNewBoard(
	w http.ResponseWriter, r *http.Request, bi ib0.IBNewBoardInfo) (
	err error, code int) {

	if bi.Name != "test" {
		return nil, 0
	} else {
		return errors.New("board already exists"), http.StatusConflict
	}
}

func (IBProviderDemo) IBPostNewThread(
	w http.ResponseWriter, r *http.Request,
	f form.Form, board string) (
	rInfo postedInfo, err error, _ int) {

	return commonNewPost(w, r, f, board, "", false)
}

func (IBProviderDemo) IBPostNewReply(
	w http.ResponseWriter, r *http.Request,
	f form.Form, board, thread string) (
	rInfo postedInfo, err error, _ int) {

	return commonNewPost(w, r, f, board, thread, true)
}

func (IBProviderDemo) IBUpdateBoard(
	w http.ResponseWriter, r *http.Request, bi ib0.IBNewBoardInfo) (
	err error, code int) {

	if bi.Name == "test" {
		return nil, 0
	} else {
		return errors.New("board not found"), http.StatusNotFound
	}
}

func (IBProviderDemo) IBDeleteBoard(
	w http.ResponseWriter, r *http.Request, board string) (
	err error, code int) {

	if board == "test" {
		return nil, 0
	} else {
		return errors.New("board not found"), http.StatusNotFound
	}
}

func (IBProviderDemo) IBDeletePost(
	w http.ResponseWriter, r *http.Request, board, post string) (
	err error, code int) {

	if board != "test" {
		return errors.New("board not found"), http.StatusNotFound
	}
	if len(post) < 4 || post[:4] != "0123" {
		return errors.New("post not found"), http.StatusNotFound
	}
	return nil, 0
}

var _ ib0.IBWebPostProvider = IBProviderDemo{}
