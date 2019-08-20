package psqlib

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/text/unicode/norm"

	au "nksrv/lib/asciiutils"
	"nksrv/lib/date"
	"nksrv/lib/emime"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/fstore"
	ht "nksrv/lib/hashtools"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mail/form"
	"nksrv/lib/mailib"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
	ib0 "nksrv/lib/webib0"
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

func (sp *PSQLIB) IBGetPostParams() (
	*form.ParserParams, form.FileOpener, []string) {

	tfields := []string{
		ib0.IBWebFormTextTitle,
		ib0.IBWebFormTextName,
		ib0.IBWebFormTextMessage,
		ib0.IBWebFormTextOptions,
	}
	if sp.webcaptcha != nil {
		tfields = append(tfields, sp.webcaptcha.TextFields()...)
	}
	return &sp.fpp, sp.ffo, tfields
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

func checkFileLimits(slimits *submissionLimits, reply bool, f form.Form) (err error, c int) {
	var onesz, allsz int64
	for _, fieldname := range FileFields {
		files := f.Files[fieldname]
		c += len(files)
		if c > int(slimits.FileMaxNum) {
			err = errTooMuchFiles(slimits.FileMaxNum)
			return
		}
		for i := range files {
			onesz = files[i].Size
			if slimits.FileMaxSizeSingle > 0 && onesz > slimits.FileMaxSizeSingle {
				err = errTooBigFileSingle(slimits.FileMaxSizeSingle)
				return
			}

			allsz += onesz
			if slimits.FileMaxSizeAll > 0 && allsz > slimits.FileMaxSizeAll {
				err = errTooBigFileAll(slimits.FileMaxSizeAll)
				return
			}

			if !allowedFileName(files[i].FileName, slimits, reply) {
				err = errFileTypeNotAllowed
				return
			}
		}
	}
	if c < int(slimits.FileMinNum) {
		err = errNotEnoughFiles(slimits.FileMinNum)
		return
	}
	return
}

func checkSubmissionLimits(slimits *submissionLimits, reply bool,
	f form.Form, mInfo mailib.MessageInfo) (err error, c int) {

	err, c = checkFileLimits(slimits, reply, f)
	if err != nil {
		return
	}

	if len(mInfo.Title) > int(slimits.MaxTitleLength) {
		err = errTooLongTitle
		return
	}
	if len(mInfo.Author) > int(slimits.MaxNameLength) {
		err = errTooLongName
		return
	}
	if len(mInfo.Message) > int(slimits.MaxMessageLength) {
		err = errTooLongMessage(slimits.MaxMessageLength)
		return
	}

	return
}

func (sp *PSQLIB) applyInstanceSubmissionLimits(
	slimits *submissionLimits, reply bool, board string) {

	// TODO

	// hardcoded instance limits, TODO make configurable

	if slimits.MaxTitleLength == 0 || slimits.MaxTitleLength > maxSubjectSize {
		slimits.MaxTitleLength = maxSubjectSize
	}

	if slimits.MaxNameLength == 0 || slimits.MaxNameLength > maxNameSize {
		slimits.MaxNameLength = maxNameSize
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
	f *os.File, ct string, fi mailib.FileInfo) (
	_ mailib.FileInfo, ext string, err error) {

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

	ext = fu.SafeExt(oname)

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
		ext = emime.MIMEPreferedExtension(ext)
		s += "." + ext
	}

	fi.ID = s
	fi.ContentType = ctype
	// yeh this is actually possible
	if oname == "" {
		fi.Original = s
	}

	return fi, ext, err
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
		if FI[i].Type.Normal() {
			FC++
		}
	}
	return
}

func (sp *PSQLIB) commonNewPost(
	w http.ResponseWriter, r *http.Request,
	f form.Form, board, thread string, isReply bool) (
	rInfo postedInfo, err error, _ int) {

	var pInfo mailib.PostInfo

	type thumbMove struct{ from, to string }
	var thumbMoves []thumbMove

	var msgfn string

	defer func() {
		if err != nil {
			f.RemoveAll()
			for _, mov := range thumbMoves {
				os.Remove(mov.from)
			}
			if msgfn != "" {
				os.Remove(msgfn)
			}
		}
	}()

	fntitle := ib0.IBWebFormTextTitle
	fnname := ib0.IBWebFormTextName
	fnmessage := ib0.IBWebFormTextMessage
	fnoptions := ib0.IBWebFormTextOptions

	// XXX more fields
	if len(f.Values[fntitle]) > 1 ||
		len(f.Values[fnname]) != 1 ||
		len(f.Values[fnmessage]) != 1 ||
		len(f.Values[fnoptions]) > 1 {

		return rInfo, errInvalidSubmission, http.StatusBadRequest
	}

	xftitle := ""
	if len(f.Values[fntitle]) != 0 {
		xftitle = f.Values[fntitle][0]
	}
	xfname := f.Values[fnname][0]
	xfmessage := f.Values[fnmessage][0]
	xfoptions := ""
	if len(f.Values[fnoptions]) != 0 {
		xfoptions = f.Values[fnoptions][0]
	}

	if sp.webcaptcha != nil {
		var code int
		if err, code = sp.webcaptcha.CheckCaptcha(w, r, f.Values); err != nil {
			return rInfo, err, code
		}
	}

	sp.log.LogPrintf(DEBUG,
		"post: board %q thread %q xftitle %q xfmessage %q xfoptions %q",
		board, thread, xftitle, xfmessage, xfoptions)

	if !utf8.ValidString(xftitle) ||
		!utf8.ValidString(xfname) ||
		!utf8.ValidString(xfmessage) ||
		!utf8.ValidString(xfoptions) {

		return rInfo, errBadSubmissionEncoding, http.StatusBadRequest
	}

	if !readableText(xftitle) ||
		!readableText(xfname) ||
		!readableText(xfmessage) ||
		!readableText(xfoptions) {

		return rInfo, errBadSubmissionChars, http.StatusBadRequest
	}

	var jbPL xtypes.JSONText // board post limits
	var jbXL xtypes.JSONText // board newthread/reply limits
	var jtRL xtypes.JSONText // thread reply limits
	var jbTO xtypes.JSONText // board threads options
	var jtTO xtypes.JSONText // thread options
	var bid boardID
	var tid sql.NullInt64
	var ref sql.NullString
	var opdate pq.NullTime

	var postLimits submissionLimits
	threadOpts := defaultThreadOptions

	// get info about board, its limits and shit. does it even exists?
	if !isReply {

		// new thread

		//sp.log.LogPrintf(DEBUG, "executing commonNewPost board query:\n%s\n", q)

		err = sp.st_prep[st_web_prepost_newthread].
			QueryRow(board).
			Scan(&bid, &jbPL, &jbXL)
		if err != nil {
			if err == sql.ErrNoRows {
				return rInfo, errNoSuchBoard, http.StatusNotFound
			}
			return rInfo,
				sp.sqlError("board row query scan", err),
				http.StatusInternalServerError
		}

		sp.log.LogPrintf(DEBUG,
			"got bid(%d) post_limits(%q) newthread_limits(%q)",
			bid, jbPL, jbXL)

		rInfo.Board = board

		postLimits = defaultNewThreadSubmissionLimits

	} else {

		// new post

		//sp.log.LogPrintf(DEBUG, "executing board x thread query:\n%s\n", q)

		err = sp.st_prep[st_web_prepost_newpost].
			QueryRow(board, thread).
			Scan(
				&bid, &jbPL, &jbXL, &tid, &jtRL,
				&jbTO, &jtTO, &ref, &opdate)
		if err != nil {
			if err == sql.ErrNoRows {
				return rInfo, errNoSuchBoard, http.StatusNotFound
			}
			return rInfo,
				sp.sqlError("board x thread row query scan", err),
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

	ok, postOpts := parsePostOptions(optimiseFormLine(xfoptions))
	if !ok {
		return rInfo, errors.New("invalid options"), http.StatusBadRequest
	}

	if postOpts.nolimit {
		// TODO check whether poster is privileged or something
		postLimits = maxSubmissionLimits
	}

	// apply instance-specific limit tweaks
	sp.applyInstanceSubmissionLimits(&postLimits, isReply, board)

	// use normalised forms
	// theorically, normalisation could increase size sometimes, which could lead to rejection of previously-fitting message
	// but it's better than accepting too big message, as that could lead to bad things later on
	pInfo.MI.Title = strings.TrimSpace(optimiseFormLine(xftitle))

	pInfo.MI.Author = strings.TrimSpace(optimiseFormLine(xfname))

	var signkeyseed []byte
	if i := strings.IndexByte(pInfo.MI.Author, '#'); i >= 0 {
		tripstr := pInfo.MI.Author[i+1:]
		// strip stuff to not leak secrets
		pInfo.MI.Author = strings.TrimSpace(pInfo.MI.Author[:i])

		// we currently only support ed25519 seed syntax
		tripseed, e := hex.DecodeString(tripstr)
		if e != nil || len(tripseed) != ed25519.SeedSize {
			return rInfo,
				errors.New("invalid tripcode syntax; we expected 64 hex chars"),
				http.StatusBadRequest
		}
		signkeyseed = tripseed
	}

	pInfo.MI.Message = tu.NormalizeTextMessage(xfmessage)

	sp.log.LogPrintf(DEBUG,
		"form fields after processing: Title(%q) Message(%q)",
		pInfo.MI.Title, pInfo.MI.Message)

	pInfo.MI.Sage = isReply &&
		(postOpts.sage || strings.ToLower(pInfo.MI.Title) == "sage")

	// check for specified limits
	var filecount int
	err, filecount = checkSubmissionLimits(&postLimits, isReply, f, pInfo.MI)
	if err != nil {
		return rInfo, err, http.StatusBadRequest
	}

	// disallow content-less msgs
	if len(pInfo.MI.Message) == 0 &&
		filecount == 0 &&
		(len(signkeyseed) == 0 || len(pInfo.MI.Title) == 0) {

		return rInfo,
			errors.New("posting empty messages isn't allowed"),
			http.StatusBadRequest
	}

	// time awareness
	tu := date.NowTimeUnix()
	// yeah we intentionally strip nanosec part
	pInfo.Date = date.UnixTimeUTC(tu)
	// could happen if OP' time is too far into the future
	// or our time too far into the past
	// result would be invalid so disallow
	if isReply && pInfo.Date.Before(opdate.Time) {
		err = errors.New(
			"time error: server's time too far into the past or thread's time too far into the future")
		return rInfo, err, http.StatusInternalServerError
	}

	// at this point message should be checked
	// we should calculate proper file names here
	// should we move files before or after writing to database?
	// maybe we should update database in 2 stages, first before, and then after?
	// or maybe we should keep journal to ensure consistency after crash?
	// decision: first write to database, then to file system. on crash, scan files table and check if files are in place (by fid).
	// there still can be the case where there are left untracked files in file system. they could be manually scanned, and damage is low.

	srcdir := sp.src.Main()
	thumbdir := sp.thm.Main()

	tplan := sp.pickThumbPlan(isReply, pInfo.MI.Sage)

	// process files
	pInfo.FI = make([]mailib.FileInfo, filecount)
	x := 0
	sp.log.LogPrint(DEBUG, "processing form files")
	for _, fieldname := range FileFields {
		files := f.Files[fieldname]
		for i := range files {
			pInfo.FI[x].Original = files[i].FileName
			pInfo.FI[x].Size = files[i].Size

			var ext string
			pInfo.FI[x], ext, err = generateFileConfig(
				files[i].F, files[i].ContentType, pInfo.FI[x])
			if err != nil {
				return rInfo, err, http.StatusInternalServerError
			}

			// thumbnail and close file
			var res thumbnailer.ThumbResult
			var tfi thumbnailer.FileInfo
			res, tfi, err = sp.thumbnailer.ThumbProcess(
				files[i].F, ext, files[i].ContentType, tplan.ThumbConfig)
			if err != nil {
				return rInfo, fmt.Errorf("error thumbnailing file: %v", err),
					http.StatusInternalServerError
			}

			pInfo.FI[x].Type = tfi.Kind
			if tfi.DetectedType != "" {
				pInfo.FI[x].ContentType = tfi.DetectedType
				// XXX change
			}
			if res.FileName != "" {
				tfile := pInfo.FI[x].ID + "." + tplan.Name + "." + res.FileExt
				pInfo.FI[x].Thumb = tfile
				pInfo.FI[x].ThumbAttrib.Width = uint32(res.Width)
				pInfo.FI[x].ThumbAttrib.Height = uint32(res.Height)
				thumbMoves = append(thumbMoves,
					thumbMove{from: res.FileName, to: thumbdir + tfile})
			}
			if len(tfi.Attrib) != 0 {
				pInfo.FI[x].FileAttrib = tfi.Attrib
			}

			for xx := 0; xx < x; xx++ {
				if pInfo.FI[xx].Equivalent(pInfo.FI[x]) {
					return rInfo,
						fmt.Errorf("duplicate file: %d is same as %d", xx, x),
						http.StatusBadRequest
				}
			}

			x++
		}
	}

	// is control message?
	isctlgrp := board == "ctl"

	// process references
	refs, inreplyto, failrefs, err := sp.processReferencesOnPost(
		sp.db.DB, pInfo.MI.Message, bid, postID(tid.Int64))
	if err != nil {
		return rInfo, err, http.StatusInternalServerError
	}
	pInfo.A.References = refs

	if isctlgrp {
		// do not add In-Reply-To for moderation messages
		inreplyto = nil
	}

	// fill in layout/sign
	var fmsgids FullMsgIDStr
	var fref FullMsgIDStr
	cref := CoreMsgIDStr(ref.String)
	if cref != "" {
		fref = FullMsgIDStr(fmt.Sprintf("<%s>", cref))
	}
	var pubkeystr string
	pInfo, fmsgids, msgfn, pubkeystr, err = sp.fillWebPostDetails(
		pInfo, f, board, fref, inreplyto, true, tu, signkeyseed)
	if err != nil {
		return rInfo, err, http.StatusInternalServerError
	}

	if fmsgids == "" {
		// lets think of Message-ID there
		fmsgids = mailib.NewRandomMessageID(tu, sp.instance)
	}

	// frontend sign
	if sp.webFrontendKey != nil {
		pInfo.H["X-Frontend-PubKey"] =
			mail.OneHeaderVal(
				hex.EncodeToString(sp.webFrontendKey[32:]))
		signature :=
			ed25519.Sign(
				sp.webFrontendKey, unsafeStrToBytes(string(fmsgids)))
		pInfo.H["X-Frontend-Signature"] =
			mail.OneHeaderVal(
				hex.EncodeToString(signature))
		// XXX store key
	}

	pInfo.MessageID = cutMsgID(fmsgids)

	// Post ID
	pInfo.ID = mailib.HashPostID_SHA1(fmsgids)

	// number of attachments
	pInfo.FC = countRealFiles(pInfo.FI)

	// start transaction
	tx, err := sp.db.DB.Begin()
	if err != nil {
		err = sp.sqlError("webpost tx begin", err)
		return rInfo, err, http.StatusInternalServerError
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if len(pInfo.FI) > 0 {
		err = sp.lockFilesTable(tx)
		if err != nil {
			return rInfo, err, http.StatusInternalServerError
		}
	}

	var modid int64
	var priv ModPriv
	if isctlgrp && pubkeystr != "" {
		sp.log.LogPrintf(DEBUG, "REGMOD %s start", pubkeystr)

		modid, priv, err = sp.registeredMod(tx, pubkeystr)
		if err != nil {
			return rInfo, err, http.StatusInternalServerError
		}

		sp.log.LogPrintf(DEBUG, "REGMOD %s done", pubkeystr)
	}

	if priv > ModPrivNone {
		// always assume we'll need exclusive lock to have consistent locking
		err = sp.preModLockFiles(tx)
	} else if len(pInfo.FI) > 0 {
		err = sp.lockFilesTable(tx)
	}
	if err != nil {
		return rInfo, err, http.StatusInternalServerError
	}

	var gpid, bpid postID
	var duplicate bool
	// perform insert
	if !isReply {
		sp.log.LogPrint(DEBUG, "inserting newthread post data to database")
		gpid, bpid, duplicate, err =
			sp.insertNewThread(tx, bid, pInfo, isctlgrp, modid)
	} else {
		sp.log.LogPrint(DEBUG, "inserting reply post data to database")
		gpid, bpid, duplicate, err =
			sp.insertNewReply(
				tx, replyTargetInfo{bid, postID(tid.Int64), threadOpts.BumpLimit},
				pInfo, modid)
	}
	if err != nil {
		return rInfo, err, http.StatusInternalServerError
	}
	if duplicate {
		// shouldn't really happen there
		err = errDuplicateArticle
		return rInfo, err, http.StatusInternalServerError
	}

	// execute mod cmd
	if priv > ModPrivNone {
		// we should execute it
		// we never put message in file when processing message

		// msgid deletion state
		var delmsgids delMsgIDState
		defer sp.cleanDeletedMsgIDs(delmsgids)

		sp.log.LogPrintf(DEBUG, "EXECMOD %s start", pInfo.MessageID)

		delmsgids, err =
			sp.execModCmd(
				tx, gpid, bid, bpid, modid, priv, pInfo, nil, pInfo.MessageID,
				cref, delmsgids)
		if err != nil {
			return rInfo, err, http.StatusInternalServerError
		}

		sp.log.LogPrintf(DEBUG, "EXECMOD %s done", pInfo.MessageID)
	}

	// fixup references
	err = sp.fixupFailRefsInTx(
		tx, gpid, failrefs, pInfo.ID, board, pInfo.MessageID)
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
			to := srcdir + pInfo.FI[x].ID
			sp.log.LogPrintf(DEBUG, "renaming %q -> %q", from, to)
			xe := fu.RenameNoClobber(from, to)
			if xe != nil {
				if os.IsExist(xe) {
					//sp.log.LogPrintf(DEBUG, "failed to rename %q to %q: %v", from, to, xe)
				} else {
					err = fmt.Errorf("failed to rename %q to %q: %v", from, to, xe)
					sp.log.LogPrint(ERROR, err.Error())
					return rInfo, err, http.StatusInternalServerError
				}
				// if failed to move, remove
				files[i].Remove()
			}
			x++
		}
	}
	if msgfn != "" {
		to := srcdir + pInfo.FI[x].ID
		sp.log.LogPrintf(DEBUG, "renaming msg %q -> %q", msgfn, to)
		xe := fu.RenameNoClobber(msgfn, to)
		if xe != nil {
			if !os.IsExist(xe) {
				err = fmt.Errorf("failed to rename %q to %q: %v", msgfn, to, xe)
				sp.log.LogPrint(ERROR, err.Error())
				return rInfo, err, http.StatusInternalServerError
			}
			// if failed to move, remove
			os.Remove(msgfn)
		}
		x++
	}
	if x != len(pInfo.FI) {
		panic(fmt.Errorf(
			"file number mismatch: have %d should have %d",
			x, len(pInfo.FI)))
	}

	// move thumbnails
	for x := range thumbMoves {
		from := thumbMoves[x].from
		to := thumbMoves[x].to

		sp.log.LogPrintf(DEBUG, "thm renaming %q -> %q", from, to)
		xe := fu.RenameNoClobber(from, to)
		if xe != nil {
			if os.IsExist(xe) {
				//sp.log.LogPrintf(DEBUG, "failed to rename %q to %q: %v", from, to, xe)
			} else {
				err = fmt.Errorf("failed to rename %q to %q: %v", from, to, xe)
				sp.log.LogPrint(ERROR, err.Error())
				return rInfo, err, http.StatusInternalServerError
			}
			os.Remove(from)
		}
	}

	// commit
	sp.log.LogPrintf(DEBUG, "webpost commit start")
	err = tx.Commit()
	if err != nil {
		err = sp.sqlError("webpost tx commit", err)
		return rInfo, err, http.StatusInternalServerError
	}
	sp.log.LogPrintf(DEBUG, "webpost commit done")

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
	w http.ResponseWriter, r *http.Request, bi ib0.IBNewBoardInfo) (
	err error, code int) {

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
	w http.ResponseWriter, r *http.Request,
	f form.Form, board string) (
	rInfo postedInfo, err error, _ int) {

	return sp.commonNewPost(w, r, f, board, "", false)
}

func (sp *PSQLIB) IBPostNewReply(
	w http.ResponseWriter, r *http.Request,
	f form.Form, board, thread string) (
	rInfo postedInfo, err error, _ int) {

	return sp.commonNewPost(w, r, f, board, thread, true)
}

func (sp *PSQLIB) IBUpdateBoard(
	w http.ResponseWriter, r *http.Request, bi ib0.IBNewBoardInfo) (
	err error, code int) {

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
	w http.ResponseWriter, r *http.Request, board string) (
	err error, code int) {

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
		return
	}

	return nil, 0
}

func (sp *PSQLIB) IBDeletePost(
	w http.ResponseWriter, r *http.Request, board, post string) (
	err error, code int) {

	// TODO
	return nil, 0
}

var _ ib0.IBWebPostProvider = (*PSQLIB)(nil)
