package psqlib

import (
	crand "crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	xtypes "github.com/jmoiron/sqlx/types"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/text/unicode/norm"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/date"
	fu "nekochan/lib/fileutil"
	"nekochan/lib/fstore"
	. "nekochan/lib/logx"
	"nekochan/lib/mail/form"
)

type formFileOpener struct {
	*fstore.FStore
}

var _ form.FileOpener = formFileOpener{}

func (o formFileOpener) OpenFile() (*os.File, error) {
	return o.FStore.TempFile("webpost-", "")
}

// FIXME: this probably in future should go thru some sort of abstractation

func (sp *PSQLIB) GetPostParams() (*form.ParserParams, form.FileOpener) {
	return &sp.fpp, sp.ffo
}

var FileFields = []string{
	"file", "file2", "file3", "file4",
	"file5", "file6", "file7", "file8",
	"file9", "file10", "file11", "file12",
	"file13", "file14", "file15", "file16",
}

func matchExtension(fn, ext string) bool {
	return len(fn) > len(ext) &&
		au.EndsWithFoldString(fn, ext) &&
		fn[len(fn)-len(ext)-1] == '.'
}

func allowedFileName(fname string, battrib *boardAttributes) bool {
	tlimits := &battrib.ThreadLimits

	if strings.IndexByte(fname, '.') < 0 {
		// we care only about extension anyway so fix that if theres none
		fname = "."
	}
	iffound := tlimits.ExtWhitelist
	var list []string
	if !tlimits.ExtWhitelist {
		list = tlimits.ExtDeny
	} else {
		list = tlimits.ExtAllow
	}
	for _, e := range list {
		if matchExtension(fname, e) {
			return iffound
		}
	}
	return !iffound
}

func checkFileLimits(battrib *boardAttributes, f form.Form) (_ error, c int) {
	tlimits := &battrib.ThreadLimits

	var onesz, allsz uint64
	for _, fieldname := range FileFields {
		files := f.Files[fieldname]
		c += len(files)
		if tlimits.FileMaxNum != 0 && c > int(tlimits.FileMaxNum) {
			return errTooMuchFiles, 0
		}
		for i := range files {
			onesz = uint64(files[i].Size)
			if tlimits.FileMaxSizeSingle != 0 &&
				onesz > tlimits.FileMaxSizeSingle {

				return errTooBigFileSingle, 0
			}

			allsz += onesz
			if tlimits.FileMaxSizeAll != 0 && allsz > tlimits.FileMaxSizeAll {
				return errTooBigFileAll, 0
			}

			if !allowedFileName(files[i].FileName, battrib) {
				return errFileTypeNotAllowed, 0
			}
		}
	}
	return
}

type postInfo struct {
	ID        string // message identifier, hash of MessageID
	MessageID string // globally unique message identifier
	Date      time.Time
	Title     string
	Author    string
	Trip      string
	Message   string
	Sage      bool
}

func checkThreadLimits(battrib *boardAttributes,
	f form.Form, pInfo postInfo) (_ error, c int) {

	tlimits := &battrib.ThreadLimits

	var e error
	e, c = checkFileLimits(battrib, f)
	if e != nil {
		return e, 0
	}

	if len(pInfo.Title) > int(tlimits.MaxTitleLength) {
		return errTooLongTitle, 0
	}
	if len(pInfo.Message) > int(tlimits.MaxMessageLength) {
		return errTooLongMessage, 0
	}

	return
}

func (sp *PSQLIB) applyInstanceThreadLimits(
	battrib *boardAttributes,
	board string, r *http.Request) {

	tlimits := &battrib.ThreadLimits

	// TODO

	// hardcoded instance limits, TODO make configurable
	const maxTitleLength = 256
	if tlimits.MaxTitleLength == 0 || tlimits.MaxTitleLength > maxTitleLength {
		tlimits.MaxTitleLength = maxTitleLength
	}

	const maxMessageLength = 32 * 1024
	if tlimits.MaxMessageLength == 0 ||
		tlimits.MaxMessageLength > maxMessageLength {

		tlimits.MaxMessageLength = maxMessageLength
	}
}

var lowerBase32Set = "abcdefghijklmnopqrstuvwxyz234567"
var lowerBase32Enc = base32.
	NewEncoding(lowerBase32Set).
	WithPadding(base32.NoPadding)

func makeInternalFileName(f *os.File, fname string) (s string, e error) {
	var h hash.Hash
	h, e = blake2s.New256([]byte(nil))
	if e != nil {
		return
	}

	_, e = io.Copy(h, f)
	if e != nil {
		return
	}

	var b [32]byte
	sum := h.Sum(b[:0])
	s = lowerBase32Enc.EncodeToString(sum)

	// append extension, if any
	if i := strings.LastIndexByte(fname, '.'); i >= 0 && i+1 < len(fname) {
		// TODO de-duplicate equivalent extensions (jpeg->jpg)?
		s += strings.ToLower(fname[i:]) // append extension including dot
	}

	return
}

type fileInfo struct {
	ID       string // storename
	Thumb    string // thumbnail
	Original string // original file name
}

type postedInfo struct {
	Board     string
	ThreadID  string
	PostID    string
	MessageID string // XXX will we actually use this for anything??
}

func (sp *PSQLIB) newMessageID(t int64) string {
	var b [8]byte
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

	var r [8]byte
	crand.Read(r[:])

	return base64.RawURLEncoding.EncodeToString(b[:]) + "." +
		base64.RawURLEncoding.EncodeToString(r[:]) + "@" + sp.instance
}

func todoHashPostID(coremsgid string) string {
	b := sha1.Sum(unsafeStrToBytes("<" + coremsgid + ">"))
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
	s = strings.Replace(msg, "\r", "", -1) // CRLF -> LF
	s = norm.NFC.String(s)
	// TODO see if below is actually needed
	//// if it ends with single newline after non-newline character, cut it out
	//if len(s) > 1 && s[len(s)-1] == '\n' && s[len(s)-2] != '\n' {
	//	s = s[:len(s)-1]
	//}
	// TODO we could process it a bit more but atm im lazy
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

func (sp *PSQLIB) commonNewPost(
	r *http.Request, f form.Form, board, thread string) (
	rInfo postedInfo, err error, _ int) {

	defer func() {
		if err != nil {
			f.RemoveAll()
		}
	}()

	// XXX more fields
	if len(f.Values["title"]) != 1 ||
		len(f.Values["message"]) != 1 {

		return rInfo, errInvalidSubmission, http.StatusBadRequest
	}

	xftitle := f.Values["title"][0]
	xfmessage := f.Values["message"][0]
	if !validFormText(xftitle) ||
		!validFormText(xfmessage) {

		return rInfo, errBadSubmissionEncoding, http.StatusBadRequest
	}

	var jcfg, jcfg2 xtypes.JSONText
	var bid boardID
	var tid postID
	var pid postID

	// get info about board, its limits and shit. does it even exists?
	if thread == "" {
		// new thread
		q := "SELECT attrib,bid FROM ib0.boards WHERE bname=$1"
		sp.log.LogPrintf(DEBUG, "executing board attrib query:\n%s\n", q)
		err = sp.db.DB.QueryRow(q, board).Scan(&jcfg, &bid)
		if err != nil {
			if err == sql.ErrNoRows {
				return rInfo, errNoSuchBoard, http.StatusNotFound
			}
			return rInfo, sp.sqlError("board row query scan", err),
				http.StatusInternalServerError
		}
		sp.log.LogPrintf(DEBUG, "got battrib(%q) bid(%d)", jcfg, bid)
	} else {
		// new post
		/* q := `WITH
			ba AS (
				SELECT attrib,bid
				FROM ib0.boards
				WHERE bname=$1
				LIMIT 1
			),
			ta AS (
				SELECT ba.attrib,ba.bid,ts.attrib,ts.tid
				FROM ba
				LEFT JOIN ib0.threads ts
				ON ba.bid=ts.bid
				WHERE tname=$2
			)
		SELECT * FROM ta` */
		q := `SELECT xb.attrib,xb.bid,xt.attrib,xt.tid
FROM ib0.boards xb
LEFT JOIN ib0.threads xt USING (bid)
WHERE xb.bname=$1 AND xt.tname=$2`
		sp.log.LogPrintf(DEBUG, "executing board x thread attrib query:\n%s\n", q)
		err = sp.db.DB.QueryRow(q, board, thread).Scan(&jcfg, &bid, &jcfg2, &tid)
		if err != nil {
			if err == sql.ErrNoRows {
				return rInfo, errNoSuchBoard, http.StatusNotFound
			}
			return rInfo, sp.sqlError("board x thread row query scan", err),
				http.StatusInternalServerError
		}
		sp.log.LogPrintf(DEBUG, "got battrib(%q) bid(%d) tattrib(%q) tid(%d)",
			jcfg, bid, jcfg2, tid)
		rInfo.Board = board
		if tid == 0 {
			return rInfo, errNoSuchThread, http.StatusNotFound
		}
	}

	battrs := defaultBoardAttributes
	err = jcfg.Unmarshal(&battrs)
	if err != nil {
		return postedInfo{}, sp.sqlError("board attr json unmarshal", err),
			http.StatusInternalServerError
	}

	var tattrs threadAttributes
	if thread != "" {
		tattrs = defaultThreadAttributes
		err = jcfg2.Unmarshal(&tattrs)
		if err != nil {
			return rInfo, sp.sqlError("thread attr json unmarshal", err),
				http.StatusInternalServerError
		}
	}

	// apply instance-specific limit tweaks
	// TODO utilise tattrs
	sp.applyInstanceThreadLimits(&battrs, board, r)

	// use normalised forms
	// theorically, normalisation could increase size sometimes, which could lead to rejection of previously-fitting message
	// but it's better than accepting too big message, as that could lead to bad things later on
	var pInfo postInfo
	pInfo.Title = strings.TrimSpace(optimiseFormLine(xftitle))
	pInfo.Message = optimiseTextMessage(xfmessage)

	// check for specified limits
	var filecount int
	// TODO utilise tattrs
	err, filecount = checkThreadLimits(&battrs, f, pInfo)
	if err != nil {
		return rInfo, err, http.StatusBadRequest
	}

	// XXX abort for empty msg if len(fmessage) == 0 && filecount == 0?

	// at this point message should be checked
	// we should calculate proper file names here
	// should we move files before or after writing to database?
	// maybe we should update database in 2 stages, first before, and then after?
	// or maybe we should keep journal to ensure consistency after crash?
	// decision: first write to database, then to file system. on crash, scan files table and check if files are in place (by fid).
	// there still can be the case where there are left untracked files in file system. they could be manually scanned, and damage is low.

	// process files
	fileInfos := make([]fileInfo, filecount)
	x := 0
	sp.log.LogPrint(DEBUG, "processing form files")
	for _, fieldname := range FileFields {
		files := f.Files[fieldname]
		for i := range files {
			orig := files[i].FileName

			var newfn string
			newfn, err = makeInternalFileName(files[i].F, orig)
			if err != nil {
				return rInfo, err, http.StatusInternalServerError
			}

			// close file, as we won't read from it directly anymore
			err = files[i].F.Close()
			if err != nil {
				return rInfo, err, http.StatusInternalServerError
			}

			// TODO extract metadata, make thumbnails here

			fileInfos[x].ID = newfn
			fileInfos[x].Original = orig

			x++
		}
	}

	tu := date.NowTimeUnix()
	pInfo.Date = date.UnixTimeUTC(tu) // yeah we intentionally strip nanosec part
	// lets think of post ID there
	pInfo.MessageID = sp.newMessageID(tu)
	pInfo.ID = todoHashPostID(pInfo.MessageID)

	// perform insert
	if thread == "" {
		sp.log.LogPrint(DEBUG, "inserting newthread post data to database")
		tid, err = sp.insertNewThread(bid, pInfo, fileInfos)
	} else {
		sp.log.LogPrint(DEBUG, "inserting reply post data to database")
		pid, err = sp.insertNewReply(bid, tid, pInfo, fileInfos)
		_ = pid // fuk u go
	}
	if err != nil {
		return rInfo, err, http.StatusInternalServerError
	}

	// move files
	sp.log.LogPrint(DEBUG, "moving form temporary files to their intended place")
	x = 0
	for _, fieldname := range FileFields {
		files := f.Files[fieldname]
		for i := range files {
			from := files[i].F.Name()
			to := sp.src.Main() + fileInfos[x].ID
			sp.log.LogPrintf(DEBUG, "renaming %q -> %q", from, to)
			xe := fu.RenameNoClobber(from, to)
			if xe != nil {
				if os.IsExist(xe) {
					sp.log.LogPrintf(DEBUG, "failed to rename %q to %q: %v", from, to, xe)
				} else {
					sp.log.LogPrintf(ERROR, "failed to rename %q to %q: %v", from, to, xe)
				}
				files[i].Remove()
			}
		}
	}

	if thread == "" {
		rInfo.ThreadID = pInfo.ID
	} else {
		rInfo.ThreadID = thread
	}
	rInfo.PostID = pInfo.ID
	rInfo.MessageID = pInfo.MessageID
	return
}

func (sp *PSQLIB) PostNewThread(
	r *http.Request, f form.Form, board string) (
	rInfo postedInfo, err error, _ int) {

	return sp.commonNewPost(r, f, board, "")
}

func (sp *PSQLIB) PostNewReply(
	r *http.Request, f form.Form, board, thread string) (
	rInfo postedInfo, err error, _ int) {

	return sp.commonNewPost(r, f, board, thread)
}
