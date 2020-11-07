package psqlib

import (
	"database/sql"
	"errors"
	"net/http"
	"os"

	ib0 "nksrv/lib/app/webib0"
	"nksrv/lib/mail/form"
	. "nksrv/lib/utils/logx"
)

// TODO make this file less messy

// FIXME: this probably in future should go thru some sort of abstractation

func (sp *PSQLIB) IBGetPostParams() (
	*form.ParserParams, form.FileOpener, func(string) bool) {

	return &sp.FPP, sp.FFO, sp.TextPostParamFunc
}

type postedInfo = ib0.IBPostedInfo

func badWebRequest(err error) error {
	return &ib0.WebPostError{Err: err, Code: http.StatusBadRequest}
}

func webNotFound(err error) error {
	return &ib0.WebPostError{Err: err, Code: http.StatusNotFound}
}

/*
 * request processing:
 * 1. validate correctness of input data, extract it
 * 2. quick db query for info based on some of input data, possibly reject there
 * 3. expensive processing of message data depending on both board info and input data (like hashing, thumbnailing)
 * 4. transaction: insert data, do sql actions; if transaction fails, retry
 * 5. somewhere, move files/thumbs in.
 *   doing that after tx commits isnt completely sound,
 *   and may only result in having excess files (could be mitigated by periodic checks)
 *   or initial unavailability (could be mitigated by exponential delays after failures);
 *   I think that's better than alternative of doing it before tx, which could lead to files being nuked after tx fails to commit;
 *   in idea we could copy over data before tx and then delete tmp files after tx, but copies are more expensive.
 *   We could use two-phase commits (PREPARE TRANSACTION) maybe, but there are some limitations with them so not yet.
 */

func wp_err_cleanup(ctx *postWebContext) {
	ctx.f.RemoveAll()
	for _, mov := range ctx.thumbInfos {
		os.Remove(mov.FullTmpName)
	}
	if ctx.msgfn != "" {
		os.Remove(ctx.msgfn)
	}
}

func wp_comm_cleanup(ctx *postWebContext) {
	var err error
	// NOTE: don't use os.RemoveAll, as we don't need traces of failed stuff gone
	err = os.Remove(ctx.src_pending)
	if err != nil {
		ctx.log.LogPrintf(
			WARN,
			"failed to remove pending src folder %q: %v",
			ctx.src_pending)
	}
	err = os.Remove(ctx.thm_pending)
	if err != nil {
		ctx.log.LogPrintf(
			WARN,
			"failed to remove pending thm folder %q: %v",
			ctx.thm_pending)
	}
}

// step #1: premature extraction and sanity validation of input data
func (sp *PSQLIB) wp_validateAndExtract(
	ctx *postWebContext, w http.ResponseWriter, r *http.Request) (err error) {

	// do text inputs processing/checking
	ctx.xf, err = sp.processTextFields(ctx.f)
	if err != nil {
		err = badWebRequest(err)
		return
	}

	// web captcha checking
	if sp.webcaptcha != nil {
		var code int
		if err, code = sp.webcaptcha.CheckCaptcha(w, r, ctx.f.Values); err != nil {
			err = &ib0.WebPostError{Err: err, Code: code}
			return
		}
	}

	var ok bool
	ok, ctx.postOpts = parsePostOptions(optimiseFormLine(ctx.xf.options))
	if !ok {
		err = badWebRequest(errInvalidOptions)
		return
	}
}

// step #2: extraction from DB
func (sp *PSQLIB) wp_dbcheck(ctx *postWebContext) (err error) {
	ctx.rInfo, ctx.wp_dbinfo, err = sp.getPrePostInfo(nil, ctx.btr, ctx.postOpts)
	return
}

func (sp *PSQLIB) commonNewPost(
	w http.ResponseWriter, r *http.Request, ctx *postWebContext) (
	rInfo postedInfo, err error) {

	defer func() {
		if err != nil {
			wp_err_cleanup(ctx)
		}
		wp_comm_cleanup(ctx)
	}()

	err = sp.wp_validateAndExtract(ctx, w, r)
	if err != nil {
		return
	}

	err = sp.wp_dbcheck(ctx)
	if err != nil {
		return
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
		NewsGroup:      "",
		Description:    "",
		ThreadsPerPage: 10,
		MaxActivePages: 10,
		MaxPages:       15,
	}
}

func (sp *PSQLIB) addNewBoard(
	bi ib0.IBNewBoardInfo) (err error, duplicate bool) {

	if bi.NewsGroup == "" {
		bi.NewsGroup = bi.Name
	}

	q := `INSERT INTO
	ib0.boards (
		b_name,
		newsgroup,
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
		$2,
		NOW(),
		$3,
		$4,
		$5,
		$6,
		$7
	)
ON CONFLICT
	DO NOTHING
RETURNING
	b_id`

	var bid boardID
	e := sp.db.DB.
		QueryRow(
			q, bi.Name, bi.NewsGroup, bi.Description,
			bi.ThreadsPerPage, bi.MaxActivePages, bi.MaxPages,
			defaultThreadOptions.BumpLimit).
		Scan(&bid)

	if e != nil {
		if e == sql.ErrNoRows {
			duplicate = true
			err = errors.New("such board already exists")
			return
		}
		err = sp.SQLError("board insertion query row scan", e)
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
		err = sp.SQLError("board update query row scan", e)
		return
	}
	aff, e := res.RowsAffected()
	if e != nil {
		err = sp.SQLError("board update query result check", e)
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
		err = sp.SQLError("board delete query row scan", e)
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
