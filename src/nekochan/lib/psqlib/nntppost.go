package psqlib

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"

	xtypes "github.com/jmoiron/sqlx/types"

	"nekochan/lib/bufreader"
	. "nekochan/lib/logx"
	"nekochan/lib/mail"
	"nekochan/lib/mailib"
	mm "nekochan/lib/minimail"
	"nekochan/lib/nntp"
)

func validMsgID(s FullMsgIDStr) bool {
	return nntp.ValidMessageID(unsafeStrToBytes(string(s)))
}

func reservedMsgID(s FullMsgIDStr) bool {
	return nntp.ReservedMessageID(unsafeStrToBytes(string(s)))
}

func cutMsgID(s FullMsgIDStr) CoreMsgIDStr {
	return CoreMsgIDStr(unsafeBytesToStr(
		nntp.CutMessageID(unsafeStrToBytes(string(s)))))
}

func (sp *PSQLIB) acceptArticleHead(
	board string, troot FullMsgIDStr) (
	ins insertSqlInfo, err error, unexpected bool) {

	// TODO ability to autoadd group?

	var jbPL xtypes.JSONText // board post limits
	var jbXL xtypes.JSONText // board newthread/reply limits
	var jtRL xtypes.JSONText // thread reply limits
	var jbTO xtypes.JSONText // board threads options
	var jtTO xtypes.JSONText // thread options

	ins.isReply = troot != ""

	ins.threadOpts = defaultThreadOptions

	// get info about board, if reply, also thread, its limits and shit.
	// does it even exists?
	if !ins.isReply {

		// new thread
		q := `SELECT bid,post_limits,newthread_limits
FROM ib0.boards
WHERE bname=$1`

		sp.log.LogPrintf(DEBUG, "executing acceptArticleHead board query:\n%s\n", q)

		err = sp.db.DB.QueryRow(q, board).Scan(&ins.bid, &jbPL, &jbXL)
		if err != nil {
			if err == sql.ErrNoRows {
				err = errNoSuchBoard
			} else {
				unexpected = true
				err = sp.sqlError("board row query scan", err)
			}
			return
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
		q := `WITH
xb AS (
	SELECT bid,post_limits,reply_limits,thread_opts
	FROM ib0.boards
	WHERE bname=$1
	LIMIT 1
)
SELECT xb.bid,xb.post_limits,xb.reply_limits,
	xtp.tid,xtp.reply_limits,xb.thread_opts,xtp.thread_opts,xtp.title
FROM xb
LEFT JOIN (
	SELECT xt.bid,xt.tid,xt.reply_limits,xt.thread_opts,xp.title
	FROM ib0.threads xt
	JOIN xb
	ON xb.bid = xt.bid
	JOIN ib0.posts xp
	ON xb.bid=xp.bid AND xt.tid=xp.tid
	WHERE xp.msgid=$2
	LIMIT 1
) AS xtp
ON xb.bid=xtp.bid`

		sp.log.LogPrintf(DEBUG, "executing board x thread query:\n%s\n", q)

		var xtid sql.NullInt64
		var xsubject sql.NullString

		err = sp.db.DB.QueryRow(q, board, string(mm.CutMessageIDStr(troot))).
			Scan(&ins.bid, &jbPL, &jbXL, &xtid, &jtRL, &jbTO, &jtTO, &xsubject)
		if err != nil {
			if err == sql.ErrNoRows {
				err = errNoSuchBoard
			} else {
				unexpected = true
				err = sp.sqlError("board x thread row query scan", err)
			}
			return
		}

		/*
			sp.log.LogPrintf(DEBUG,
				"got bid(%d) b.post_limits(%q) b.reply_limits(%q) tid(%#v) "+
					"t.reply_limits(%q) b.thread_opts(%q) t.thread_opts(%q) p.msgid(%q)",
				ins.bid, jbPL, jbXL, xtid, jtRL, jbTO, jtTO)
		*/

		if xtid.Int64 <= 0 {
			// TODO ability to put such messages elsewhere?
			err = errNoSuchThread
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

func (sp *PSQLIB) nntpCheckArticleExists(
	unsafe_sid CoreMsgIDStr) (exists bool, err error) {

	var dummy int
	q := "SELECT 1 FROM ib0.posts WHERE msgid = $1 LIMIT 1"
	err = sp.db.DB.QueryRow(q, string(unsafe_sid)).Scan(&dummy)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, sp.sqlError("article existence query scan", err)
		}
		return false, nil
	} else {
		return true, nil
	}
}

var (
	nntpIncomingTempDir = "_tin"
	nntpIncomingDir     = "_in"
)

func (sp *PSQLIB) nntpSendIncomingArticle(
	name string, H mail.Headers, info nntpParsedInfo) {

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

	w.ResSendArticleToBePosted()
	r := ro.OpenReader()
	err, unexpected := sp.netnewsHandleSubmissionDirectly(r)
	if err != nil {
		if !unexpected {
			w.ResPostingFailed(err)
		} else {
			w.ResInternalError(err)
		}
		r.Discard(-1)
	} else {
		w.ResPostingAccepted()
	}
	return true
}

func (sp *PSQLIB) netnewsHandleSubmissionDirectly(
	r io.Reader) (err error, unexpected bool) {

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

	info, err, unexpected := sp.nntpDigestTransferHead(mh.H, "", true)
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
	w Responder, cs *ConnState, ro nntp.ReaderOpener, msgid CoreMsgID) bool {

	var err error

	unsafe_sid := unsafeCoreMsgIDToStr(msgid)

	// check if we already have it
	exists, err := sp.nntpCheckArticleExists(unsafe_sid)
	if err != nil {
		w.ResInternalError(err)
		return true
	}
	if exists {
		// article exists, false for default message
		return false
	}

	w.ResSendArticleToBeTransferred()
	r := ro.OpenReader()

	info, newname, H, err, unexpected :=
		sp.handleIncoming(r, unsafe_sid, nntpIncomingDir)
	if err != nil {
		if !unexpected {
			w.ResTransferRejected(err)
		} else {
			w.ResInternalError(err)
		}
		r.Discard(-1)
		return true
	}

	sp.nntpSendIncomingArticle(newname, H, info)

	// we're done there, signal success
	w.ResTransferSuccess()
	return true
}

// + ok: 238{ResArticleWanted} fail: 431{ResArticleWantLater} 438{ResArticleNotWanted[false]}
func (sp *PSQLIB) HandleCheck(
	w Responder, cs *ConnState, msgid CoreMsgID) bool {

	var err error

	unsafe_sid := unsafeCoreMsgIDToStr(msgid)

	// check if we already have it
	exists, err := sp.nntpCheckArticleExists(unsafe_sid)
	if err != nil {
		w.ResInternalError(err)
		return true
	}
	if exists {
		// article exists, false for default message
		return false
	}
	w.ResArticleWanted(msgid)
	return true
}

// + ok: 239{ResArticleTransferedOK} 439{ResArticleRejected[false]}
func (sp *PSQLIB) HandleTakeThis(
	w Responder, cs *ConnState, r nntp.ArticleReader, msgid CoreMsgID) bool {

	var err error

	unsafe_sid := unsafeCoreMsgIDToStr(msgid)
	// check if we already have it
	exists, err := sp.nntpCheckArticleExists(unsafe_sid)
	if err != nil {
		w.ResInternalError(err)
		r.Discard(-1)
		return true
	}
	if exists {
		// article exists, false for default message
		return false
	}

	info, newname, H, err, unexpected :=
		sp.handleIncoming(r, unsafe_sid, nntpIncomingDir)
	if err != nil {
		if !unexpected {
			w.ResArticleRejected(msgid, err)
		} else {
			w.ResInternalError(err)
		}
		r.Discard(-1)
		return true
	}

	sp.nntpSendIncomingArticle(newname, H, info)

	// we're done there, signal success
	w.ResArticleTransferedOK(msgid)
	return true
}

func (sp *PSQLIB) handleIncoming(
	r io.Reader, unsafe_sid CoreMsgIDStr, incdir string) (
	info nntpParsedInfo, newname string, H mail.Headers,
	err error, unexpected bool) {

	info, f, H, err, unexpected :=
		sp.handleIncomingIntoFile(r, unsafe_sid)
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
	r io.Reader, unsafe_sid CoreMsgIDStr) (
	info nntpParsedInfo, f *os.File, H mail.Headers,
	err error, unexpected bool) {

	var mh mail.MessageHead
	mh, err = mail.ReadHeaders(r, mailib.DefaultHeaderSizeLimit)
	if err != nil {
		err = fmt.Errorf("failed reading headers: %v", err)
		return
	}
	defer mh.Close()

	info, err, unexpected = sp.nntpDigestTransferHead(mh.H, unsafe_sid, false)
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
	msgid CoreMsgIDStr) (err error, unexpected bool) {

	// check if we already have it
	exists, e := sp.nntpCheckArticleExists(msgid)
	if e != nil {
		err = fmt.Errorf(
			"error while checking article existence: %v", e)
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
	H mail.Headers, B *bufreader.BufReader) (
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
