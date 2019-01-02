package psqlib

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	xtypes "github.com/jmoiron/sqlx/types"
	"golang.org/x/text/unicode/norm"

	au "centpd/lib/asciiutils"
	"centpd/lib/date"
	"centpd/lib/emime"
	fu "centpd/lib/fileutil"
	"centpd/lib/fstore"
	ht "centpd/lib/hashtools"
	. "centpd/lib/logx"
	"centpd/lib/mail/form"
	"centpd/lib/mailib"
	tu "centpd/lib/textutils"
	ib0 "centpd/lib/webib0"
)

// TODO make this file less messy

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
	if len(mInfo.Author) > int(slimits.MaxNameLength) {
		return errTooLongName, 0
	}
	if len(mInfo.Message) > int(slimits.MaxMessageLength) {
		return errTooLongMessage, 0
	}

	return
}

func (sp *PSQLIB) applyInstanceSubmissionLimits(
	slimits *submissionLimits, reply bool, board string) {

	// TODO

	// hardcoded instance limits, TODO make configurable

	if slimits.MaxTitleLength == 0 || slimits.MaxTitleLength > maxSubjectLen {
		slimits.MaxTitleLength = maxSubjectLen
	}

	if slimits.MaxNameLength == 0 || slimits.MaxNameLength > maxNameLen {
		slimits.MaxNameLength = maxNameLen
	}

	const maxMessageLength = mailib.DefaultMaxTextLen
	if slimits.MaxMessageLength == 0 ||
		slimits.MaxMessageLength > maxMessageLength {

		slimits.MaxMessageLength = maxMessageLength
	}
}

func (sp *PSQLIB) applyInstanceThreadOptions(
	threadOpts *threadOptions, board string) {

	// TODO
}

// expects file to be seeked at 0
func generateFileConfig(
	f *os.File, ct string, fi mailib.FileInfo) (_ mailib.FileInfo, err error) {

	hash, hashtype, err := ht.MakeFileHash(f)
	if err != nil {
		return
	}
	s := hash + "-" + hashtype

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
		s += "." + emime.MIMEPreferedExtension(ext)
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
	"\t", " ",
	"\000", "")

func optimiseFormLine(line string) (s string) {
	s = lineReplacer.Replace(line)
	s = norm.NFC.String(s)
	return
}

func countRealFiles(FI []mailib.FileInfo) (FC int) {
	for i := range FI {
		if FI[i].Type != mailib.FTypeMsg {
			FC++
		}
	}
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
	fnname := ib0.IBWebFormTextName
	fnmessage := ib0.IBWebFormTextMessage

	// XXX more fields
	if len(f.Values[fntitle]) != 1 ||
		len(f.Values[fnname]) != 1 ||
		len(f.Values[fnmessage]) != 1 {

		return rInfo, errInvalidSubmission, http.StatusBadRequest
	}

	xftitle := f.Values[fntitle][0]
	xfname := f.Values[fnname][0]
	xfmessage := f.Values[fnmessage][0]

	sp.log.LogPrintf(DEBUG,
		"post: board %q thread %q xftitle %q xfmessage %q",
		board, thread, xftitle, xfmessage)

	if !validFormText(xftitle) ||
		!validFormText(xfname) ||
		!validFormText(xfmessage) {

		return rInfo, errBadSubmissionEncoding, http.StatusBadRequest
	}

	var jbPL xtypes.JSONText // board post limits
	var jbXL xtypes.JSONText // board newthread/reply limits
	var jtRL xtypes.JSONText // thread reply limits
	var jbTO xtypes.JSONText // board threads options
	var jtTO xtypes.JSONText // thread options
	var bid boardID
	var tid sql.NullInt64
	var ref sql.NullString

	var postLimits submissionLimits
	threadOpts := defaultThreadOptions

	// get info about board, its limits and shit. does it even exists?
	if !isReply {

		// new thread
		q := `SELECT b_id,post_limits,newthread_limits
FROM ib0.boards
WHERE b_name=$1`

		//sp.log.LogPrintf(DEBUG, "executing commonNewPost board query:\n%s\n", q)

		err = sp.db.DB.QueryRow(q, board).Scan(&bid, &jbPL, &jbXL)
		if err != nil {
			if err == sql.ErrNoRows {
				return rInfo, errNoSuchBoard, http.StatusNotFound
			}
			return rInfo, sp.sqlError("board row query scan", err),
				http.StatusInternalServerError
		}

		sp.log.LogPrintf(DEBUG, "got bid(%d) post_limits(%q) newthread_limits(%q)",
			bid, jbPL, jbXL)

		rInfo.Board = board

		postLimits = defaultNewThreadSubmissionLimits

	} else {

		// new post
		// TODO count files to enforce limit. do not bother about atomicity, too low cost/benefit ratio
		q := `WITH
	xb AS (
		SELECT b_id,post_limits,reply_limits,thread_opts
		FROM ib0.boards
		WHERE b_name=$1
		LIMIT 1
	)
SELECT
	xb.b_id,xb.post_limits,xb.reply_limits,
	xtp.t_id,xtp.reply_limits,xb.thread_opts,xtp.thread_opts,xtp.msgid
FROM
	xb
LEFT JOIN
	(
		SELECT
			xt.b_id,xt.t_id,xt.reply_limits,xt.thread_opts,xp.msgid
		FROM
			ib0.threads xt
		JOIN
			xb
		ON
			xb.b_id = xt.b_id
		JOIN
			ib0.bposts xbp
		ON
			xt.b_id=xbp.b_id AND xt.t_id=xbp.b_p_id
		JOIN
			ib0.posts xp
		ON
			xbp.g_p_id = xp.g_p_id
		WHERE
			xt.t_name=$2
		LIMIT
			1
	) AS xtp
ON
	xb.b_id=xtp.b_id`

		//sp.log.LogPrintf(DEBUG, "executing board x thread query:\n%s\n", q)

		err = sp.db.DB.QueryRow(q, board, thread).Scan(
			&bid, &jbPL, &jbXL, &tid, &jtRL, &jbTO, &jtTO, &ref)
		if err != nil {
			if err == sql.ErrNoRows {
				return rInfo, errNoSuchBoard, http.StatusNotFound
			}
			return rInfo, sp.sqlError("board x thread row query scan", err),
				http.StatusInternalServerError
		}

		sp.log.LogPrintf(DEBUG,
			"got bid(%d) b.post_limits(%q) b.reply_limits(%q) tid(%#v) "+
				"t.reply_limits(%q) b.thread_opts(%q) t.thread_opts(%q) p.msgid(%#v)",
			bid, jbPL, jbXL, tid, jtRL, jbTO, jtTO, ref)

		rInfo.Board = board

		if tid.Int64 <= 0 {
			return rInfo, errNoSuchThread, http.StatusNotFound
		}

		rInfo.ThreadID = thread

		postLimits = defaultReplySubmissionLimits

	}

	err = sp.unmarshalBoardConfig(&postLimits, jbPL, jbXL)
	if err != nil {
		return rInfo, err, http.StatusInternalServerError
	}

	if isReply {
		err = sp.unmarshalThreadConfig(
			&postLimits, &threadOpts, jtRL, jbTO, jtTO)
		if err != nil {
			return rInfo, err, http.StatusInternalServerError
		}

		sp.applyInstanceThreadOptions(&threadOpts, board)
	}

	// apply instance-specific limit tweaks
	sp.applyInstanceSubmissionLimits(&postLimits, isReply, board)

	// use normalised forms
	// theorically, normalisation could increase size sometimes, which could lead to rejection of previously-fitting message
	// but it's better than accepting too big message, as that could lead to bad things later on
	pInfo.MI.Title = strings.TrimSpace(optimiseFormLine(xftitle))

	pInfo.MI.Author = strings.TrimSpace(optimiseFormLine(xfname))
	if i := strings.IndexByte(pInfo.MI.Author, '#'); i >= 0 {
		// TODO tripcode processing
		// for now it's better to just strip stuff to not leak secrets
		pInfo.MI.Author = strings.TrimSpace(pInfo.MI.Author[:i])
	}

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

	// process references
	refs, inreplyto, err := sp.processReferencesOnPost(
		pInfo.MI.Message, bid, postID(tid.Int64))
	if err != nil {
		return rInfo, err, http.StatusInternalServerError
	}
	pInfo.A.References = refs

	// XXX abort for empty msg if len(fmessage) == 0 && filecount == 0?

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

	// fill in info about post
	tu := date.NowTimeUnix()
	pInfo.Date = date.UnixTimeUTC(tu) // yeah we intentionally strip nanosec part

	pInfo = sp.fillWebPostDetails(pInfo, board,
		CoreMsgIDStr(ref.String), inreplyto)

	// lets think of Message-ID there
	fmsgids := mailib.NewRandomMessageID(tu, sp.instance)
	pInfo.MessageID = cutMsgID(fmsgids)

	// Post ID
	pInfo.ID = mailib.HashPostID_SHA1(fmsgids)

	// number of attachments
	pInfo.FC = countRealFiles(pInfo.FI)

	// perform insert
	if !isReply {
		sp.log.LogPrint(DEBUG, "inserting newthread post data to database")
		_, err = sp.insertNewThread(bid, pInfo)
	} else {
		sp.log.LogPrint(DEBUG, "inserting reply post data to database")
		_, err = sp.insertNewReply(replyTargetInfo{
			bid, postID(tid.Int64), threadOpts.BumpLimit}, pInfo)
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
					//sp.log.LogPrintf(DEBUG, "failed to rename %q to %q: %v", from, to, xe)
				} else {
					sp.log.LogPrintf(ERROR, "failed to rename %q to %q: %v", from, to, xe)
				}
				files[i].Remove()
			}
			x++
		}
	}
	if x != len(pInfo.FI) {
		panic(fmt.Errorf(
			"file number mismatch: have %d should have %d",
			x, len(pInfo.FI)))
	}

	if !isReply {
		rInfo.ThreadID = pInfo.ID
	}
	rInfo.PostID = pInfo.ID
	rInfo.MessageID = pInfo.MessageID
	return
}

func (sp *PSQLIB) IBDefaultBoardInfo() ib0.IBNewBoardInfo {
	return ib0.IBNewBoardInfo{
		Name:           "",
		Description:    "",
		ThreadsPerPage: 10,
		MaxActivePages: 10,
		MaxPages:       15,
	}
}

func (sp *PSQLIB) addNewBoard(
	bi ib0.IBNewBoardInfo) (err error, duplicate bool) {

	q := `INSERT INTO
	ib0.boards (
		b_name,
		badded,
		bdesc,
		threads_per_page,
		max_active_pages,
		max_pages
	)
VALUES
	(
		$1,
		NOW(),
		$2,
		$3,
		$4,
		$5
	)
ON CONFLICT
	DO NOTHING
RETURNING
	b_id`

	var bid boardID
	e := sp.db.DB.QueryRow(q, bi.Name, bi.Description,
		bi.ThreadsPerPage, bi.MaxActivePages, bi.MaxPages).Scan(&bid)
	if e != nil {
		if e == sql.ErrNoRows {
			duplicate = true
			err = errors.New("such board already exists")
			return
		}
		err = sp.sqlError("board insertion query row scan", e)
		return
	}
	return nil, false
}

func (sp *PSQLIB) IBPostNewBoard(
	r *http.Request, bi ib0.IBNewBoardInfo) (err error, code int) {

	err, duplicate := sp.addNewBoard(bi)
	if err != nil {
		if duplicate {
			code = http.StatusConflict
		} else {
			code = http.StatusInternalServerError
		}
		return
	}
	return nil, 0
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

func (sp *PSQLIB) IBUpdateBoard(
	r *http.Request, bi ib0.IBNewBoardInfo) (err error, code int) {

	q := `UPDATE ib0.boards
SET
	bdesc = $2,
	threads_per_page = $3,
	max_active_pages = $4,
	max_pages = $5
WHERE bname = $1`
	res, e := sp.db.DB.Exec(q, bi.Name, bi.Description,
		bi.ThreadsPerPage, bi.MaxActivePages, bi.MaxPages)
	if e != nil {
		err = sp.sqlError("board update query row scan", e)
		code = http.StatusInternalServerError
		return
	}
	aff, e := res.RowsAffected()
	if e != nil {
		err = sp.sqlError("board update query result check", e)
		code = http.StatusInternalServerError
		return
	}
	if aff == 0 {
		err = errors.New("no such board")
		code = http.StatusNotFound
		return
	}
	return nil, 0
}

func (sp *PSQLIB) IBDeleteBoard(
	r *http.Request, board string) (err error, code int) {

	// TODO delet any of posts in board
	var bid boardID
	q := `DELETE FROM ib0.boards WHERE b_name=$1 RETURNING bid`
	e := sp.db.DB.QueryRow(q, board).Scan(&bid)
	if e != nil {
		if e == sql.ErrNoRows {
			return errors.New("no such board"), http.StatusNotFound
		}
		err = sp.sqlError("board delete query row scan", e)
		code = http.StatusInternalServerError
	}
	return nil, 0
}

func (sp *PSQLIB) IBDeletePost(
	r *http.Request, board, post string) (err error, code int) {

	// TODO
	return nil, 0
}

var _ ib0.IBWebPostProvider = (*PSQLIB)(nil)
