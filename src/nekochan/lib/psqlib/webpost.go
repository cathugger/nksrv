package psqlib

import (
	crand "crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	xtypes "github.com/jmoiron/sqlx/types"
	"golang.org/x/text/unicode/norm"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/date"
	"nekochan/lib/emime"
	fu "nekochan/lib/fileutil"
	"nekochan/lib/fstore"
	ht "nekochan/lib/hashtools"
	. "nekochan/lib/logx"
	"nekochan/lib/mail/form"
	"nekochan/lib/mailib"
	tu "nekochan/lib/textutils"
	ib0 "nekochan/lib/webib0"
)

var FileFields = ib0.IBWebFormFileFields

type formFileOpener struct {
	*fstore.FStore
}

var _ form.FileOpener = formFileOpener{}

func (o formFileOpener) OpenFile() (*os.File, error) {
	return o.FStore.TempFile("webpost-", "")
}

// FIXME: this probably in future should go thru some sort of abstractation

func (sp *PSQLIB) IBGetPostParams() (*form.ParserParams, form.FileOpener) {
	return &sp.fpp, sp.ffo
}

func matchExtension(fn, ext string) bool {
	return len(fn) > len(ext) &&
		au.EndsWithFoldString(fn, ext) &&
		fn[len(fn)-len(ext)-1] == '.'
}

func allowedFileName(fname string, slimits *submissionLimits, reply bool) bool {
	if strings.IndexByte(fname, '.') < 0 {
		// we care only about extension anyway so fix that if theres none
		fname = "."
	}
	iffound := slimits.ExtWhitelist
	var list []string
	if !slimits.ExtWhitelist {
		list = slimits.ExtDeny
	} else {
		list = slimits.ExtAllow
	}
	for _, e := range list {
		if matchExtension(fname, e) {
			return iffound
		}
	}
	return !iffound
}

func checkFileLimits(slimits *submissionLimits, reply bool, f form.Form) (_ error, c int) {
	var onesz, allsz uint64
	for _, fieldname := range FileFields {
		files := f.Files[fieldname]
		c += len(files)
		if slimits.FileMaxNum != 0 && c > int(slimits.FileMaxNum) {
			return errTooMuchFiles, 0
		}
		for i := range files {
			onesz = uint64(files[i].Size)
			if slimits.FileMaxSizeSingle != 0 &&
				onesz > slimits.FileMaxSizeSingle {

				return errTooBigFileSingle, 0
			}

			allsz += onesz
			if slimits.FileMaxSizeAll != 0 && allsz > slimits.FileMaxSizeAll {
				return errTooBigFileAll, 0
			}

			if !allowedFileName(files[i].FileName, slimits, reply) {
				return errFileTypeNotAllowed, 0
			}
		}
	}
	return
}

func checkSubmissionLimits(slimits *submissionLimits, reply bool,
	f form.Form, mInfo mailib.MessageInfo) (_ error, c int) {

	var e error
	e, c = checkFileLimits(slimits, reply, f)
	if e != nil {
		return e, 0
	}

	if len(mInfo.Title) > int(slimits.MaxTitleLength) {
		return errTooLongTitle, 0
	}
	if len(mInfo.Message) > int(slimits.MaxMessageLength) {
		return errTooLongMessage, 0
	}

	return
}

func (sp *PSQLIB) applyInstanceSubmissionLimits(
	slimits *submissionLimits, reply bool,
	board string, r *http.Request) {

	// TODO

	// hardcoded instance limits, TODO make configurable
	const maxTitleLength = 256
	if slimits.MaxTitleLength == 0 || slimits.MaxTitleLength > maxTitleLength {
		slimits.MaxTitleLength = maxTitleLength
	}

	const maxMessageLength = 32 * 1024
	if slimits.MaxMessageLength == 0 ||
		slimits.MaxMessageLength > maxMessageLength {

		slimits.MaxMessageLength = maxMessageLength
	}
}

func (sp *PSQLIB) applyInstanceThreadOptions(threadOpts *threadOptions,
	board string, r *http.Request) {

	// TODO
}

// expects file to be seeked at 0
func generateFileConfig(
	f *os.File, ct string, fi mailib.FileInfo) (_ mailib.FileInfo, err error) {

	s, err := ht.MakeFileHash(f)
	if err != nil {
		return
	}

	// prefer info from file name, try figuring out content-type from it
	// if that fails, try looking into content-type, try figure out filename
	// if both fail, just use given type and given filename

	// append extension, if any
	oname := fi.Original
	ext := ""
	if i := strings.LastIndexByte(oname, '.'); i >= 0 && i+1 < len(oname) {
		ext = oname[i+1:]
	}
	ctype := emime.MIMECanonicalTypeByExtension(ext)
	if ctype == "" && ct != "" {
		mexts, e := emime.MIMEExtensionsByType(ct)
		if e == nil {
			if len(mexts) != 0 {
				ext = mexts[0]
			}
		} else {
			// bad ct
			ct = ""
		}
	}
	if ctype == "" {
		if ct != "" {
			ctype = ct
		} else {
			ctype = "application/octet-stream"
		}
	}

	if len(ext) != 0 {
		s += "." + ext
	}

	fi.ID = s
	fi.ContentType = ctype
	// yeh this is actually possible
	if oname == "" {
		fi.Original = s
	}

	return fi, err
}

type postedInfo = ib0.IBPostedInfo

func (sp *PSQLIB) newMessageID(t int64) string {
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

	return ht.SBase64Enc.EncodeToString(b[:]) + "." +
		ht.SBase64Enc.EncodeToString(r[:]) + "@" + sp.instance
}

// TODO: more algos
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
	r *http.Request, f form.Form, board, thread string, isReply bool) (
	rInfo postedInfo, err error, _ int) {

	var pInfo mailib.PostInfo

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

	sp.log.LogPrintf(DEBUG,
		"form fields: xftitle(%q) xfmessage(%q)", xftitle, xfmessage)

	if !validFormText(xftitle) ||
		!validFormText(xfmessage) {

		return rInfo, errBadSubmissionEncoding, http.StatusBadRequest
	}

	var jcfg [5]xtypes.JSONText
	var bid boardID
	var tid postID
	var pid postID
	var ref CoreMsgIDStr

	var postLimits submissionLimits
	threadOpts := defaultThreadOptions

	// get info about board, its limits and shit. does it even exists?
	if !isReply {

		// new thread
		q := `SELECT bid,post_limits,newthread_limits
FROM ib0.boards
WHERE bname=$1`

		sp.log.LogPrintf(DEBUG, "executing board query:\n%s\n", q)

		err = sp.db.DB.QueryRow(q, board).Scan(&bid, &jcfg[0], &jcfg[1])
		if err != nil {
			if err == sql.ErrNoRows {
				return rInfo, errNoSuchBoard, http.StatusNotFound
			}
			return rInfo, sp.sqlError("board row query scan", err),
				http.StatusInternalServerError
		}

		sp.log.LogPrintf(DEBUG, "got bid(%d) post_limits(%q) newthread_limits(%q)",
			bid, jcfg[0], jcfg[1])

		rInfo.Board = board

		postLimits = defaultNewThreadSubmissionLimits

	} else {

		// new post
		// TODO count files to enforce limit. do not bother about atomicity, too low cost/benefit ratio
		q := `SELECT xb.bid,xb.post_limits,xb.reply_limits,
	xt.tid,xt.reply_limits,xb.thread_opts,xt.thread_opts,xp.msgid
FROM ib0.boards xb
LEFT JOIN ib0.threads xt
ON xb.bid=xt.bid
INNER JOIN ib0.posts xp
ON xt.bid=xp.bid AND xt.tid=xp.pid
WHERE xb.bname=$1 AND xt.tname=$2`

		sp.log.LogPrintf(DEBUG, "executing board x thread query:\n%s\n", q)

		err = sp.db.DB.QueryRow(q, board, thread).Scan(
			&bid, &jcfg[0], &jcfg[1], &tid, &jcfg[2], &jcfg[3], &jcfg[4], &ref)
		if err != nil {
			if err == sql.ErrNoRows {
				return rInfo, errNoSuchBoard, http.StatusNotFound
			}
			return rInfo, sp.sqlError("board x thread row query scan", err),
				http.StatusInternalServerError
		}

		sp.log.LogPrintf(DEBUG,
			"got bid(%d) b.post_limits(%q) b.reply_limits(%q) tid(%d) "+
				"t.reply_limits(%q) b.thread_opts(%q) t.thread_opts(%q) p.msgid(%q)",
			bid, jcfg[0], jcfg[1], tid, jcfg[2], jcfg[3], jcfg[4], ref)

		rInfo.Board = board

		if tid == 0 {
			return rInfo, errNoSuchThread, http.StatusNotFound
		}

		rInfo.ThreadID = thread

		postLimits = defaultReplySubmissionLimits

	}

	// jcfg[0] - b.post_limits in all cases
	err = jcfg[0].Unmarshal(&postLimits)
	if err != nil {
		return rInfo, sp.sqlError("jcfg[0] json unmarshal", err),
			http.StatusInternalServerError
	}

	// jcfg[1] - either b.newthread_limits or b.reply_limits
	err = jcfg[1].Unmarshal(&postLimits)
	if err != nil {
		return rInfo, sp.sqlError("jcfg[1] json unmarshal", err),
			http.StatusInternalServerError
	}

	if isReply {
		// jcfg[2] - t.reply_limits
		err = jcfg[2].Unmarshal(&postLimits)
		if err != nil {
			return rInfo, sp.sqlError("jcfg[2] json unmarshal", err),
				http.StatusInternalServerError
		}

		// jcfg[3] - b.thread_opts
		err = jcfg[3].Unmarshal(&threadOpts)
		if err != nil {
			return rInfo, sp.sqlError("jcfg[3] json unmarshal", err),
				http.StatusInternalServerError
		}

		// jcfg[4] - t.thread_opts
		err = jcfg[4].Unmarshal(&threadOpts)
		if err != nil {
			return rInfo, sp.sqlError("jcfg[4] json unmarshal", err),
				http.StatusInternalServerError
		}

		sp.applyInstanceThreadOptions(&threadOpts, board, r)
	}

	// apply instance-specific limit tweaks
	sp.applyInstanceSubmissionLimits(&postLimits, isReply, board, r)

	// use normalised forms
	// theorically, normalisation could increase size sometimes, which could lead to rejection of previously-fitting message
	// but it's better than accepting too big message, as that could lead to bad things later on
	pInfo.MI.Title = strings.TrimSpace(optimiseFormLine(xftitle))
	pInfo.MI.Message = tu.NormalizeTextMessage(xfmessage)
	sp.log.LogPrintf(DEBUG,
		"form fields after processing: Title(%q) Message(%q)",
		pInfo.MI.Title, pInfo.MI.Message)

	// check for specified limits
	var filecount int
	err, filecount = checkSubmissionLimits(&postLimits, isReply, f, pInfo.MI)
	if err != nil {
		return rInfo, err, http.StatusBadRequest
	}

	// XXX abort for empty msg if len(fmessage) == 0 && filecount == 0?

	// fill in info about post
	tu := date.NowTimeUnix()
	pInfo.Date = date.UnixTimeUTC(tu) // yeah we intentionally strip nanosec part
	pInfo = sp.fillWebPostDetails(pInfo, board, ref)

	// at this point message should be checked
	// we should calculate proper file names here
	// should we move files before or after writing to database?
	// maybe we should update database in 2 stages, first before, and then after?
	// or maybe we should keep journal to ensure consistency after crash?
	// decision: first write to database, then to file system. on crash, scan files table and check if files are in place (by fid).
	// there still can be the case where there are left untracked files in file system. they could be manually scanned, and damage is low.

	// process files
	pInfo.FI = make([]mailib.FileInfo, filecount)
	x := 0
	sp.log.LogPrint(DEBUG, "processing form files")
	for _, fieldname := range FileFields {
		files := f.Files[fieldname]
		for i := range files {
			pInfo.FI[x].Original = files[i].FileName
			pInfo.FI[x].Size = files[i].Size

			pInfo.FI[x], err = generateFileConfig(
				files[i].F, files[i].ContentType, pInfo.FI[x])
			if err != nil {
				return rInfo, err, http.StatusInternalServerError
			}

			// close file, as we won't read from it directly anymore
			err = files[i].F.Close()
			if err != nil {
				return rInfo, fmt.Errorf("error closing file: %v", err), http.StatusInternalServerError
			}

			// TODO extract metadata, make thumbnails here

			x++
		}
	}

	// lets think of post ID there
	pInfo.MessageID = sp.newMessageID(tu)
	pInfo.ID = todoHashPostID(pInfo.MessageID)

	// perform insert
	if !isReply {
		sp.log.LogPrint(DEBUG, "inserting newthread post data to database")
		tid, err = sp.insertNewThread(bid, pInfo)
	} else {
		sp.log.LogPrint(DEBUG, "inserting reply post data to database")
		pid, err = sp.insertNewReply(
			replyTargetInfo{bid, tid, threadOpts.BumpLimit}, pInfo)
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
			to := sp.src.Main() + pInfo.FI[x].ID
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

	if !isReply {
		rInfo.ThreadID = pInfo.ID
	}
	rInfo.PostID = pInfo.ID
	rInfo.MessageID = pInfo.MessageID
	return
}

func (sp *PSQLIB) IBPostNewBoard(
	bi ib0.IBNewBoardInfo) (created bool, err error, code int) {

	q := `INSERT INTO ib0.boards (
	bname,badded,bdesc,threads_per_page,max_active_pages,max_pages)
VALUES ($1,NOW(),$2,$3,$4,$5)
ON CONFLICT DO NOTHING
RETURNING bid`
	var bid boardID
	e := sp.db.DB.QueryRow(q, bi.Name, bi.Description,
		bi.ThreadsPerPage, bi.MaxActivePages, bi.MaxPages).Scan(&bid)
	if e != nil {
		if e == sql.ErrNoRows {
			code = http.StatusConflict
			return
		}
		err = sp.sqlError("board insertion query row scan", e)
		code = http.StatusInternalServerError
		return
	}
	return true, nil, 0
}

func (sp *PSQLIB) IBPostNewThread(
	r *http.Request, f form.Form, board string) (
	rInfo postedInfo, err error, _ int) {

	return sp.commonNewPost(r, f, board, "", false)
}

func (sp *PSQLIB) IBPostNewReply(
	r *http.Request, f form.Form, board, thread string) (
	rInfo postedInfo, err error, _ int) {

	return sp.commonNewPost(r, f, board, thread, true)
}

var _ ib0.IBWebPostProvider = (*PSQLIB)(nil)
