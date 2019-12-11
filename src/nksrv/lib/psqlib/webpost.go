package psqlib

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/ed25519"

	"nksrv/lib/date"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/ibref_nntp"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mail/form"
	"nksrv/lib/mailib"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/webcaptcha"
	ib0 "nksrv/lib/webib0"
)

// TODO make this file less messy

// FIXME: this probably in future should go thru some sort of abstractation

func makePostParamFunc(c *webcaptcha.WebCaptcha) func(string) bool {
	tfields := []string{
		ib0.IBWebFormTextTitle,
		ib0.IBWebFormTextName,
		ib0.IBWebFormTextMessage,
		ib0.IBWebFormTextOptions,
	}
	if c != nil {
		tfields = append(tfields, c.TextFields()...)
	}
	return form.FieldsCheckFunc(tfields)
}

func (sp *PSQLIB) IBGetPostParams() (
	*form.ParserParams, form.FileOpener, func(string) bool) {

	return &sp.fpp, sp.ffo, sp.textPostParamFunc
}

type postedInfo = ib0.IBPostedInfo

func badWebRequest(err error) error {
	return &ib0.WebPostError{Err: err, Code: http.StatusBadRequest}
}

func webNotFound(err error) error {
	return &ib0.WebPostError{Err: err, Code: http.StatusNotFound}
}

func (sp *PSQLIB) commonNewPost(
	w http.ResponseWriter, r *http.Request,
	f form.Form, board, thread string, isReply bool) (
	rInfo postedInfo, err error) {

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

	// do text inputs processing/checking
	xf, err := sp.processTextFields(f)
	if err != nil {
		err = badWebRequest(err)
		return
	}

	// web captcha checking
	if sp.webcaptcha != nil {
		var code int
		if err, code = sp.webcaptcha.CheckCaptcha(w, r, f.Values); err != nil {
			err = &ib0.WebPostError{Err: err, Code: code}
			return
		}
	}

	ok, postOpts := parsePostOptions(optimiseFormLine(xf.options))
	if !ok {
		err = badWebRequest(errInvalidOptions)
		return
	}

	rInfo, bid, tid, ref, postLimits, opdate, err :=
		sp.getPrePostInfo(isReply, board, thread, postOpts)
	if err != nil {
		return
	}

	// use normalised forms
	// theorically, normalisation could increase size sometimes, which could lead to rejection of previously-fitting message
	// but it's better than accepting too big message, as that could lead to bad things later on
	pInfo.MI.Title = strings.TrimSpace(optimiseFormLine(xf.title))
	pInfo.MI.Author = strings.TrimSpace(optimiseFormLine(xf.name))

	var signkeyseed []byte
	if i := strings.IndexByte(pInfo.MI.Author, '#'); i >= 0 {
		tripstr := pInfo.MI.Author[i+1:]
		// strip stuff to not leak secrets
		pInfo.MI.Author = strings.TrimSpace(pInfo.MI.Author[:i])

		// we currently only support ed25519 seed syntax
		tripseed, e := hex.DecodeString(tripstr)
		if e != nil || len(tripseed) != ed25519.SeedSize {
			err = badWebRequest(errInvalidTripcode)
			return
		}
		signkeyseed = tripseed
	}

	pInfo.MI.Message = tu.NormalizeTextMessage(xf.message)
	pInfo.MI.Sage = isReply &&
		(postOpts.sage || strings.ToLower(pInfo.MI.Title) == "sage")

	// check for specified limits
	var filecount int
	err, filecount = checkSubmissionLimits(&postLimits, isReply, f, pInfo.MI)
	if err != nil {
		err = badWebRequest(err)
		return
	}

	// disallow content-less msgs
	if len(pInfo.MI.Message) == 0 &&
		filecount == 0 &&
		(len(signkeyseed) == 0 || len(pInfo.MI.Title) == 0) {

		err = badWebRequest(errEmptyMsg)
		return
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
		return
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
				return
			}

			// thumbnail and close file
			var res thumbnailer.ThumbResult
			var tfi thumbnailer.FileInfo
			res, tfi, err = sp.thumbnailer.ThumbProcess(
				files[i].F, ext, pInfo.FI[x].ContentType, tplan.ThumbConfig)
			if err != nil {
				err = fmt.Errorf("error thumbnailing file: %v", err)
				return
			}

			pInfo.FI[x].Type = tfi.Kind
			if tfi.DetectedType != "" {
				pInfo.FI[x].ContentType = tfi.DetectedType
				// XXX change
			}
			// save it
			pInfo.FI[x].Extras.ContentType = pInfo.FI[x].ContentType
			// thumbnail
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
					err = badWebRequest(
						fmt.Errorf(
							"duplicate file: %d is same as %d", xx, x))
					return
				}
			}

			x++
		}
	}

	// is control message?
	isctlgrp := board == "ctl"

	// process references
	srefs, irefs := ibref_nntp.ParseReferences(pInfo.MI.Message)
	var inreplyto []string
	// we need to build In-Reply-To beforehand
	// best-effort basis, in most cases it'll be okay
	inreplyto, err = sp.processReferencesOnPost(
		sp.db.DB, srefs, bid, postID(tid.Int64), isctlgrp)
	if err != nil {
		return
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
		return
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

	// before starting transaction, ensure stmt for postinsert is ready
	// otherwise deadlock is v possible
	var gstmt *sql.Stmt
	if !isReply {
		gstmt, err = sp.getNTStmt(len(pInfo.FI))
	} else {
		gstmt, err = sp.getNPStmt(npTuple{len(pInfo.FI), pInfo.MI.Sage})
	}
	if err != nil {
		return
	}

	// start transaction
	tx, err := sp.db.DB.Begin()
	if err != nil {
		err = sp.sqlError("webpost tx begin", err)
		return
	}
	defer func() {
		if err != nil {
			sp.log.LogPrintf(DEBUG, "webpost rollback start")
			_ = tx.Rollback()
			sp.log.LogPrintf(DEBUG, "webpost rollback done")
		}
	}()

	err = sp.makeDelTables(tx)
	if err != nil {
		return
	}

	var modid uint64
	var hascap bool
	var modCC ModCombinedCaps

	if isctlgrp && pubkeystr != "" {

		sp.log.LogPrintf(DEBUG, "REGMOD %s start", pubkeystr)

		modid, hascap, modCC, err =
			sp.registeredMod(tx, pubkeystr)
		if err != nil {
			return
		}

		sp.log.LogPrintf(DEBUG, "REGMOD %s done", pubkeystr)
	}

	var gpid, bpid postID
	var duplicate bool
	// perform insert
	if !isReply {
		sp.log.LogPrint(DEBUG, "inserting newthread post data to database")
		gpid, bpid, duplicate, err =
			sp.insertNewThread(tx, gstmt, bid, pInfo, isctlgrp, modid)
	} else {
		sp.log.LogPrint(DEBUG, "inserting reply post data to database")
		gpid, bpid, duplicate, err =
			sp.insertNewReply(
				tx, gstmt,
				replyTargetInfo{bid, postID(tid.Int64)},
				pInfo, modid)
	}
	if err != nil {
		return
	}
	if duplicate {
		// shouldn't really happen there
		err = errDuplicateArticle
		return
	}

	// execute mod cmd
	if hascap {
		// we should execute it
		// we never put message in file when processing message

		// msgid deletion state
		var delmsgids delMsgIDState
		defer func() { sp.cleanDeletedMsgIDs(delmsgids) }()

		sp.log.LogPrintf(DEBUG, "EXECMOD %s start", pInfo.MessageID)

		delmsgids, _, err, _ =
			sp.execModCmd(
				tx, gpid, bid, bpid,
				modid, modCC,
				pInfo, nil, pInfo.MessageID,
				cref, delmsgids, delModIDState{})
		if err != nil {
			return
		}

		sp.log.LogPrintf(DEBUG, "EXECMOD %s done", pInfo.MessageID)
	}

	// NOTE
	// current method may sometimes fail finding stuff in highly concurrent conditions
	// we're inside transaction, therefore we won't see messages being added in other transactions
	// messages in other thransactions also won't be able to see our new message so they wont be able to notify
	// in idea, after-tx job should find us, but only if it runs after we have commited (not guaranteed)
	// TODO implement job doing after-processing for this; initial best-effort scan can still be handy
	err = sp.processRefsAfterPost(
		tx,
		srefs, irefs, inreplyto,
		bid, uint64(tid.Int64), bpid,
		pInfo.ID, board, pInfo.MessageID)

	if err != nil {
		return
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
					return
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
				return
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
				return
			}
			os.Remove(from)
		}
	}

	// commit
	sp.log.LogPrintf(DEBUG, "webpost commit start")
	err = tx.Commit()
	if err != nil {
		err = sp.sqlError("webpost tx commit", err)
		return
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
		max_pages,
		cfg_t_bump_limit
	)
VALUES
	(
		$1,
		NOW(),
		$2,
		$3,
		$4,
		$5,
		$6
	)
ON CONFLICT
	DO NOTHING
RETURNING
	b_id`

	var bid boardID
	e := sp.db.DB.
		QueryRow(
			q, bi.Name, bi.Description,
			bi.ThreadsPerPage, bi.MaxActivePages, bi.MaxPages,
			defaultThreadOptions.BumpLimit).
		Scan(&bid)

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
	err error) {

	err, duplicate := sp.addNewBoard(bi)
	if err != nil {
		if duplicate {
			return &ib0.WebPostError{Err: err, Code: http.StatusConflict}
		}
		return
	}
	return nil
}

func (sp *PSQLIB) IBPostNewThread(
	w http.ResponseWriter, r *http.Request,
	f form.Form, board string) (
	rInfo postedInfo, err error) {

	return sp.commonNewPost(w, r, f, board, "", false)
}

func (sp *PSQLIB) IBPostNewReply(
	w http.ResponseWriter, r *http.Request,
	f form.Form, board, thread string) (
	rInfo postedInfo, err error) {

	return sp.commonNewPost(w, r, f, board, thread, true)
}

func (sp *PSQLIB) IBUpdateBoard(
	w http.ResponseWriter, r *http.Request, bi ib0.IBNewBoardInfo) (
	err error) {

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
		return
	}
	aff, e := res.RowsAffected()
	if e != nil {
		err = sp.sqlError("board update query result check", e)
		return
	}
	if aff == 0 {
		return webNotFound(errNoSuchBoard)
	}
	return nil
}

func (sp *PSQLIB) IBDeleteBoard(
	w http.ResponseWriter, r *http.Request, board string) (
	err error) {

	// TODO delet any of posts in board
	var bid boardID
	q := `DELETE FROM ib0.boards WHERE b_name=$1 RETURNING bid`
	e := sp.db.DB.QueryRow(q, board).Scan(&bid)
	if e != nil {
		if e == sql.ErrNoRows {
			return webNotFound(errNoSuchBoard)
		}
		err = sp.sqlError("board delete query row scan", e)
		return
	}

	return nil
}

func (sp *PSQLIB) IBDeletePost(
	w http.ResponseWriter, r *http.Request, board, post string) (
	err error) {

	// TODO
	return nil
}

var _ ib0.IBWebPostProvider = (*PSQLIB)(nil)
