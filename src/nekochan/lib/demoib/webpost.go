package demoib

import (
	crand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"nekochan/lib/date"
	"nekochan/lib/mail/form"
	ib0 "nekochan/lib/webib0"
)

var FileFields = ib0.IBWebFormFileFields

type formFileOpener struct {
}

var _ form.FileOpener = formFileOpener{}

func (formFileOpener) OpenFile() (*os.File, error) {
	return ioutil.TempFile("", "webpost-")
}

// FIXME: this probably in future should go thru some sort of abstractation

func (IBProviderDemo) IBGetPostParams() (*form.ParserParams, form.FileOpener) {
	return &form.DefaultParserParams, formFileOpener{}
}

// custom sort-able base64 set
var sBase64Set = "-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"
var sBase64Enc = base64.
	NewEncoding(sBase64Set).
	WithPadding(base64.NoPadding)

type postedInfo = ib0.IBPostedInfo

func newMessageID(t int64) string {
	var b [8]byte
	// TAI64
	u := uint64(t) + 4611686018427387914
	b[7] = byte(u)
	u >>= 8
	b[6] = byte(u)
	u >>= 8
	b[5] = byte(u)
	u >>= 8
	b[4] = byte(u)
	u >>= 8
	b[3] = byte(u)
	u >>= 8
	b[2] = byte(u)
	u >>= 8
	b[1] = byte(u)
	u >>= 8
	b[0] = byte(u)

	var r [12]byte
	crand.Read(r[:])

	return sBase64Enc.EncodeToString(b[:]) + "." +
		sBase64Enc.EncodeToString(r[:]) + "@test.invalid"
}

// TODO: more algos
func todoHashPostID(coremsgid string) string {
	b := sha1.Sum([]byte("<" + coremsgid + ">"))
	return hex.EncodeToString(b[:])
}

func readableText(s string) bool {
	for _, c := range s {
		if (c < 32 && c != '\n' && c != '\r' && c != '\t') || c == 127 {
			return false
		}
	}
	return true
}

func validFormText(s string) bool {
	return utf8.ValidString(s) && readableText(s)
}

func optimiseTextMessage(msg string) (s string) {
	// normalise using form C
	s = norm.NFC.String(msg)
	// trim line endings, and empty lines at the end
	lines := strings.Split(s, "\n")
	for i, v := range lines {
		lines[i] = strings.TrimRightFunc(v, unicode.IsSpace)
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			break
		}
		lines = lines[:i]
	}
	s = strings.Join(lines, "\n")
	// ensure we don't have any CR left
	s = strings.Replace(s, "\r", "", -1)
	return
}

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
	r *http.Request, f form.Form, board, thread string, isReply bool) (
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

	if !validFormText(xftitle) ||
		!validFormText(xfmessage) {

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
	pMessage := optimiseTextMessage(xfmessage)
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
	pMessageID := newMessageID(tu)
	pID := todoHashPostID(pMessageID)

	if !isReply {
		rInfo.ThreadID = pID
	}
	rInfo.PostID = pID
	rInfo.MessageID = pMessageID
	return
}

func (IBProviderDemo) IBPostNewBoard(
	bi ib0.IBNewBoardInfo) (created bool, err error, code int) {

	if bi.Name == "test" {
		return true, nil, 0
	} else {
		return false, errors.New("board already exists"), http.StatusConflict
	}
}

func (IBProviderDemo) IBPostNewThread(
	r *http.Request, f form.Form, board string) (
	rInfo postedInfo, err error, _ int) {

	return commonNewPost(r, f, board, "", false)
}

func (IBProviderDemo) IBPostNewReply(
	r *http.Request, f form.Form, board, thread string) (
	rInfo postedInfo, err error, _ int) {

	return commonNewPost(r, f, board, thread, true)
}

var _ ib0.IBWebPostProvider = IBProviderDemo{}
