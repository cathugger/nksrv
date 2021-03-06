package pipostnntp

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"time"

	xtypes "github.com/jmoiron/sqlx/types"

	"nksrv/lib/app/mailib"
	"nksrv/lib/app/psqlib/internal/pibase"
	"nksrv/lib/mail"
	"nksrv/lib/nntp"
	. "nksrv/lib/utils/logx"
	mm "nksrv/lib/utils/minimail"
	"nksrv/lib/utils/text/bufreader"
)

type (
	TFullMsgID    = nntp.TFullMsgID
	TCoreMsgID    = nntp.TCoreMsgID
	TFullMsgIDStr = nntp.TFullMsgIDStr
	TCoreMsgIDStr = nntp.TCoreMsgIDStr
)

func validMsgID(s TFullMsgIDStr) bool {
	return nntp.ValidMessageID(unsafeStrToBytes(string(s)))
}

func cutMsgID(s TFullMsgIDStr) TCoreMsgIDStr {
	return TCoreMsgIDStr(unsafeBytesToStr(
		nntp.CutMessageID(unsafeStrToBytes(string(s)))))
}

func shouldAutoAddNNTPPostGroup(sp *pibase.PSQLIB, group string) bool {
	return sp.NGPGlobal.CheckGroup(group) ||
		sp.NGPAnyServer.CheckGroup(group)
	// TODO per-server-client privileges
}

func acceptArticleHead(
	sp *pibase.PSQLIB,
	board string, troot TFullMsgIDStr, pdate int64) (
	ins insertSqlInfo, err error, unexpected bool, wantroot bool) {

	var jbPL xtypes.JSONText // board post limits
	var jbXL xtypes.JSONText // board newthread/reply limits
	var jtRL xtypes.JSONText // thread reply limits
	var jbTO xtypes.JSONText // board threads options
	var jtTO xtypes.JSONText // thread options

	ins.isReply = troot != ""

	ins.threadOpts = pibase.DefaultThreadOptions

	// get info about board, if reply, also thread, its limits and shit.
	// does it even exists?
	if !ins.isReply {

		// new thread
		q := `SELECT b_id,post_limits,newthread_limits
FROM ib0.boards
WHERE b_name=$1`

		//sp.log.LogPrintf(DEBUG, "executing acceptArticleHead board query:\n%s\n", q)

		nadd := 0
		for {
			err = sp.DB.DB.QueryRow(q, board).Scan(&ins.bid, &jbPL, &jbXL)
			if err != nil {
				if err == sql.ErrNoRows {
					if !shouldAutoAddNNTPPostGroup(sp, board) || nadd >= 20 {
						err = pibase.ErrNoSuchBoard
						return
					}
					// try adding new
				} else {
					unexpected = true
					err = sp.SQLError("board row query scan", err)
					return
				}
			} else {
				// we got board
				break
			}

			nadd++

			// try to add new board
			bi := sp.IBDefaultBoardInfo()
			bi.Name = board
			var dup bool
			err, dup = addNewBoard(sp, bi)
			if err != nil && !dup {
				unexpected = true
				err = fmt.Errorf("addNewBoard error: %v", err)
				return
			}
		}

		/*
			sp.log.LogPrintf(DEBUG,
				"got bid(%d) post_limits(%q) newthread_limits(%q)",
				ins.bid, jbPL, jbXL)
		*/

		ins.postLimits = defaultNewThreadSubmissionLimits

	} else {

		// new post
		// TODO count files to enforce limit. do not bother about atomicity, too low cost/benefit ratio
		// get info about board being posted in and thread and OP
		q := `WITH
	xb AS (
		SELECT
			b_id,post_limits,reply_limits,thread_opts
		FROM
			ib0.boards
		WHERE
			b_name=$1
		LIMIT
			1
	)
SELECT
	xb.b_id,xb.post_limits,xb.reply_limits,
	xtp.b_id,xtp.b_t_id,xtp.reply_limits,xb.thread_opts,xtp.thread_opts,
	xtp.title,xtp.date_sent
FROM
	xb
FULL JOIN (
	SELECT
		xt.b_id,xt.b_t_id,xt.reply_limits,xt.thread_opts,xp.title,xp.date_sent
	FROM
		ib0.threads xt
	JOIN
		ib0.bposts xbp
	ON
		xt.b_id=xbp.b_id AND xt.b_t_id=xbp.b_t_id
	JOIN
		ib0.gposts xp
	ON
		xbp.g_p_id = xp.g_p_id
	WHERE
		xp.msgid=$2
	LIMIT
		1
) AS xtp
ON TRUE`

		//sp.log.LogPrintf(DEBUG, "executing board x thread query:\n%s\n", q)

		var xbid sql.NullInt64
		var xtbid sql.NullInt64
		var xtid sql.NullInt64
		var xsubject sql.NullString
		var xreftime *time.Time

		err = sp.db.DB.QueryRow(q, board, string(mm.CutMessageIDStr(troot))).
			Scan(&xbid, &jbPL, &jbXL, &xtbid, &xtid, &jtRL, &jbTO, &jtTO,
				&xsubject, &xreftime)
		if err != nil {
			if err == sql.ErrNoRows {
				err = errNoSuchBoard
				// don't autoadd for replies.
				// reply obviously won't have any parent post in this board
				// because board didn't exist before.
				// ... we can actually suggest to add parent tho
				if sp.shouldAutoAddNNTPPostGroup(board) {
					wantroot = true
				}
			} else {
				unexpected = true
				err = sp.sqlError("board x thread row query scan", err)
			}
			return
		}

		ins.bid = boardID(xbid.Int64)

		/*
			sp.log.LogPrintf(DEBUG,
				"got bid(%d) b.post_limits(%q) b.reply_limits(%q) tid(%#v) "+
					"t.reply_limits(%q) b.thread_opts(%q) t.thread_opts(%q) p.msgid(%q)",
				ins.bid, jbPL, jbXL, xtid, jtRL, jbTO, jtTO)
		*/

		if xtbid.Int64 > 0 && xtid.Int64 > 0 {
			// such thread exists
			if xbid.Int64 != xtbid.Int64 {
				// but in different board...
				err = errors.New("post refers to thread in different board")
				return
			}
			// at this point everything is right
			// keep goin
		} else if xbid.Int64 <= 0 {
			// no such board exists
			err = errNoSuchBoard
			if sp.shouldAutoAddNNTPPostGroup(board) {
				wantroot = true
			}
			return
		} else {
			// no such thread exists
			err = errNoSuchThread
			wantroot = true
			return
		}

		if xreftime.Unix() > pdate {
			err = errors.New("post has date before post it refers to")
			return
		}

		ins.tid = postID(xtid.Int64)
		ins.refSubject = xsubject.String

		ins.postLimits = defaultReplySubmissionLimits

	}

	err = sp.unmarshalBoardConfig(&ins.postLimits, jbPL, jbXL)
	if err != nil {
		unexpected = true
		return
	}

	if ins.isReply {
		err = sp.unmarshalThreadConfig(
			&ins.postLimits, &ins.threadOpts, jtRL, jbTO, jtTO)
		if err != nil {
			unexpected = true
			return
		}

		sp.applyInstanceThreadOptions(&ins.threadOpts, board)
	}

	// apply instance-specific limit tweaks
	sp.applyInstanceSubmissionLimits(&ins.postLimits, ins.isReply, board)

	//sp.log.LogPrintf(DEBUG, "acceptArticleHead done")

	// done here
	return
}

func (sp *PSQLIB) nntpCheckArticleExistsOrBanned(
	unsafe_sid TCoreMsgIDStr) (exists bool, err error) {

	var dummy int64

	err = sp.st_prep[st_nntp_article_exists_or_banned_by_msgid].
		QueryRow(string(unsafe_sid)).Scan(&dummy)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, sp.sqlError("article existence query scan", err)
		}
		return false, nil
	}

	return true, nil
}

func (sp *PSQLIB) nntpCheckArticleValid(
	unsafe_sid TCoreMsgIDStr) (exists bool, err error) {

	var dummy int64

	err = sp.st_prep[st_nntp_article_valid_by_msgid].
		QueryRow(string(unsafe_sid)).Scan(&dummy)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, sp.sqlError("article existence query scan", err)
		}
		return false, nil
	}

	return true, nil
}

func (sp *PSQLIB) nntpSendIncomingArticle(
	name string, H mail.HeaderMap, info nntpParsedInfo) {

	defer os.Remove(name)

	f, err := os.Open(name)
	if err != nil {
		sp.log.LogPrintf(WARN,
			"nntpSendIncomingArticle: failed to open: %v", err)
		return
	}
	defer f.Close()

	sp.netnewsSubmitFullArticle(f, H, info)
}

func (sp *PSQLIB) HandlePost(
	w Responder, cs *ConnState, ro nntp.ReaderOpener) bool {

	nntpAbortOnErr(w.ResSendArticleToBePosted())
	r := ro.OpenReader()
	err, unexpected := sp.netnewsHandleSubmissionDirectly(r, false)
	if err != nil {
		if !unexpected {
			err = w.ResPostingFailed(err)
		} else {
			err = w.ResInternalError(err)
		}
		nntpAbortOnErr(err)
		_, _ = r.Discard(-1)
	} else {
		nntpAbortOnErr(w.ResPostingAccepted())
	}
	return true
}

func (sp *PSQLIB) netnewsHandleSubmissionDirectly(
	r io.Reader, notrace bool) (
	err error, unexpected bool) {

	lr := &io.LimitedReader{R: r, N: int64(math.MaxInt64)}

	var mh mail.MessageHead
	mh, err = mail.ReadHeaders(lr, mailib.DefaultHeaderSizeLimit)
	if err != nil {
		err = fmt.Errorf("failed reading headers: %v", err)
		return
	}
	defer mh.Close()

	limit := sp.maxArticleBodySize
	lr.N = limit + 1 - int64(len(mh.B.Buffered()))

	info, err, unexpected, _ :=
		sp.nntpDigestTransferHead(mh.H, "", "", true, notrace)
	if lr.N <= 0 {
		// limit exceeded
		err = fmt.Errorf("article body too large, up to %d allowed", limit)
		return
	}
	if err != nil {
		return
	}

	if info.FullMsgIDStr != "" {
		err, unexpected = sp.ensureArticleDoesntExist(cutMsgID(info.FullMsgIDStr))
		if err != nil {
			return
		}
	}

	return sp.netnewsSubmitArticle(mh.B, mh.H, info)
}

// + iok: 335{ResSendArticleToBeTransferred} ifail: 435{ResTransferNotWanted[false]} 436{ResTransferFailed}
// cok: 235{ResTransferSuccess} cfail: 436{ResTransferFailed} 437{ResTransferRejected}
func (sp *PSQLIB) HandleIHave(
	w Responder, cs *ConnState, ro nntp.ReaderOpener, msgid TCoreMsgID) bool {

	var err error

	unsafe_sid := unsafeCoreMsgIDToStr(msgid)

	// check if we already have it
	exists, err := sp.nntpCheckArticleExistsOrBanned(unsafe_sid)
	if err != nil {
		nntpAbortOnErr(w.ResInternalError(err))
		return true
	}
	if exists {
		// article exists, false for default message
		return false
	}

	nntpAbortOnErr(w.ResSendArticleToBeTransferred())
	r := ro.OpenReader()

	info, newname, H, err, unexpected, _ :=
		sp.handleIncoming(r, unsafe_sid, "", nntpIncomingDir, false)
	if err != nil {
		if !unexpected {
			err = w.ResTransferRejected(err)
		} else {
			err = w.ResInternalError(err)
		}
		nntpAbortOnErr(err)
		_, _ = r.Discard(-1)
		return true
	}

	sp.nntpSendIncomingArticle(newname, H, info)

	// we're done there, signal success
	nntpAbortOnErr(w.ResTransferSuccess())
	return true
}

// + ok: 238{ResArticleWanted} fail: 431{ResArticleWantLater} 438{ResArticleNotWanted[false]}
func (sp *PSQLIB) HandleCheck(
	w Responder, cs *ConnState, msgid TCoreMsgID) bool {

	var err error

	unsafe_sid := unsafeCoreMsgIDToStr(msgid)

	// check if we already have it
	exists, err := sp.nntpCheckArticleExistsOrBanned(unsafe_sid)
	if err != nil {
		nntpAbortOnErr(w.ResInternalError(err))
		return true
	}
	if exists {
		// article exists, false for default message
		return false
	}
	nntpAbortOnErr(w.ResArticleWanted(msgid))
	return true
}

// + ok: 239{ResArticleTransferedOK} 439{ResArticleRejected[false]}
func (sp *PSQLIB) HandleTakeThis(
	w Responder, cs *ConnState, r nntp.ArticleReader, msgid TCoreMsgID) bool {

	var err error

	unsafe_sid := unsafeCoreMsgIDToStr(msgid)
	// check if we already have it
	exists, err := sp.nntpCheckArticleExistsOrBanned(unsafe_sid)
	if err != nil {
		nntpAbortOnErr(w.ResInternalError(err))
		_, _ = r.Discard(-1)
		return true
	}
	if exists {
		// article exists, false for default message
		return false
	}

	info, newname, H, err, unexpected, _ :=
		sp.handleIncoming(r, unsafe_sid, "", nntpIncomingDir, false)
	if err != nil {
		if !unexpected {
			err = w.ResArticleRejected(msgid, err)
		} else {
			err = w.ResInternalError(err)
		}
		nntpAbortOnErr(err)
		_, _ = r.Discard(-1)
		return true
	}

	sp.nntpSendIncomingArticle(newname, H, info)

	// we're done there, signal success
	nntpAbortOnErr(w.ResArticleTransferedOK(msgid))
	return true
}

func (sp *PSQLIB) handleIncoming(
	r io.Reader, unsafe_sid TCoreMsgIDStr, expectgroup string, incdir string,
	notrace bool) (
	info nntpParsedInfo, newname string, H mail.HeaderMap,
	err error, unexpected bool, wantroot TFullMsgIDStr) {

	info, f, H, err, unexpected, wantroot :=
		sp.handleIncomingIntoFile(r, unsafe_sid, expectgroup, notrace)
	if err != nil {
		return
	}

	// XXX should we have option to call f.Sync()?
	// would this level of reliability be worth performance degradation?
	err = f.Close()
	if err != nil {
		err = fmt.Errorf("error writing body: %v", err)
		unexpected = true
		return
	}

	newname = path.Join(sp.nntpfs.Main()+incdir, path.Base(f.Name()))
	err = os.Rename(f.Name(), newname)
	if err != nil {
		err = sp.sqlError("incoming file move", err)
		unexpected = true
		return
	}

	return
}

func (sp *PSQLIB) handleIncomingIntoFile(
	r io.Reader, unsafe_sid TCoreMsgIDStr, expectgroup string, notrace bool) (
	info nntpParsedInfo, f *os.File, H mail.HeaderMap,
	err error, unexpected bool, wantroot TFullMsgIDStr) {

	var mh mail.MessageHead
	mh, err = mail.ReadHeaders(r, mailib.DefaultHeaderSizeLimit)
	if err != nil {
		err = fmt.Errorf("failed reading headers: %v", err)
		return
	}
	defer mh.Close()

	info, err, unexpected, wantroot =
		sp.nntpDigestTransferHead(mh.H, unsafe_sid, expectgroup, false, notrace)
	if err != nil {
		return
	}

	f, err, unexpected = sp.netnewsCopyArticleToFile(mh.H, mh.B)
	if err != nil {
		return
	}

	H = mh.H
	return
}

var errArticleAlreadyExists = errors.New("article with this Message-ID already exists")

func (sp *PSQLIB) ensureArticleDoesntExist(
	msgid TCoreMsgIDStr) (err error, unexpected bool) {

	// check if we already have it
	exists, err := sp.nntpCheckArticleExistsOrBanned(msgid)
	if err != nil {
		err = fmt.Errorf(
			"error while checking article existence: %v", err)
		unexpected = true
		return
	}
	if exists {
		// article exists, false for default message
		err = errArticleAlreadyExists
		unexpected = false
		return
	}

	return
}

func (sp *PSQLIB) netnewsCopyArticleToFile(
	H mail.HeaderMap, B *bufreader.BufReader) (
	f *os.File, err error, unexpected bool) {

	// TODO file should start with current timestamp/increasing counter
	f, err = sp.nntpfs.NewFile(nntpIncomingTempDir, "", ".eml")
	if err != nil {
		err = fmt.Errorf("error making temporary file: %v", err)
		unexpected = true
		return
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
			f = nil
		}
	}()

	err = mail.WriteHeaders(f, H, false)
	if err != nil {
		if err != mail.ErrHeaderLineTooLong {
			err = fmt.Errorf("error writing headers: %v", err)
			unexpected = true
		}
		return
	}

	_, err = fmt.Fprintf(f, "\n")
	if err != nil {
		err = fmt.Errorf("error writing newline: %v", err)
		unexpected = true
		return
	}

	limit := sp.maxArticleBodySize
	n, err := io.CopyN(f, B, limit+1)
	if n > limit {
		// limit exceeded
		err = fmt.Errorf("article body too large, up to %d allowed", limit)
		return
	}
	if err != io.EOF {
		err = fmt.Errorf("error writing body: %v", err)
		unexpected = true
		return
	}
	err = nil

	return
}
