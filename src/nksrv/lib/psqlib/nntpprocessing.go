package psqlib

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	au "nksrv/lib/asciiutils"
	"nksrv/lib/date"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/ibref_nntp"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mailib"
	"nksrv/lib/mailibsign"
	"nksrv/lib/nntp"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
)


func (ctx *nntpPostCtx) netnewsSubmitFullArticle(r io.Reader) {

	mh, err := mail.SkipHeaders(r)
	if err != nil {
		sp.log.LogPrintf(WARN,
			"netnewsSubmitFullArticle: failed skipping headers: %v", err)
		return
	}
	defer mh.Close()

	err, unexpected := ctx.netnewsSubmitArticle(mh.B)
	if err != nil {
		if !unexpected {
			sp.log.LogPrintf(WARN, "netnewsSubmitArticle: %v", err)
		} else {
			sp.log.LogPrintf(ERROR, "netnewsSubmitArticle: %v", err)
		}
	}
}


func (ctx *nntpPostCtx) netnewsSubmitArticle(
	br io.Reader) (err error, unexpected bool) {

	defer func() {
		if err != nil {
			ctx.pn_cleanup_on_err()
		}
	}()

	err, unexpected = ctx.pn_eatbody(br)
	if err != nil {
		return
	}

	// before starting transaction, ensure stmt for postinsert is ready
	// otherwise deadlock is v possible
	var gstmt *sql.Stmt
	if !info.isReply {
		gstmt, err = sp.getNTStmt(len(pi.FI))
	} else {
		gstmt, err = sp.getNPStmt(npTuple{len(pi.FI), pi.MI.Sage})
	}
	if err != nil {
		unexpected = true
		return
	}

	// start transaction
	tx, err := sp.db.DB.Begin()
	if err != nil {
		err = sp.sqlError("nntp tx begin", err)
		unexpected = true
		return
	}
	defer func() {
		if err != nil {
			sp.log.LogPrintf(DEBUG, "nntppost rollback start")
			_ = tx.Rollback()
			sp.log.LogPrintf(DEBUG, "nntppost rollback done")
		}
	}()

	err = sp.makeDelTables(tx)
	if err != nil {
		unexpected = true
		return
	}

	isctlgrp := info.Newsgroup == "ctl"

	var modid uint64
	var hascap bool
	var modCC ModCombinedCaps

	if isctlgrp && pubkeystr != "" {

		sp.log.LogPrintf(DEBUG, "REGMOD %s start", pubkeystr)

		modid, hascap, modCC, err =
			sp.registeredMod(tx, pubkeystr)
		if err != nil {
			unexpected = true
			return
		}

		sp.log.LogPrintf(DEBUG, "REGMOD %s done", pubkeystr)
	}

	var gpid, bpid postID
	var duplicate bool
	// perform insert
	if !info.isReply {
		sp.log.LogPrint(DEBUG, "inserting newthread post data to database")
		gpid, bpid, duplicate, err =
			sp.insertNewThread(tx, gstmt, info.bid, pi, isctlgrp, modid)
	} else {
		sp.log.LogPrint(DEBUG, "inserting reply post data to database")
		gpid, bpid, duplicate, err =
			sp.insertNewReply(
				tx, gstmt,
				replyTargetInfo{info.bid, info.tid},
				pi, modid)
	}
	if err != nil {
		err = fmt.Errorf("post insertion failed: %v", err)
		unexpected = true
		return
	}
	if duplicate {
		err = errDuplicateArticle
		return
	}

	// execute mod cmd
	if hascap {

		var cref CoreMsgIDStr
		if info.FRef != "" {
			cref = cutMsgID(info.FRef)
		}

		// msgid deletion state
		var delmsgids delMsgIDState
		defer func() { sp.cleanDeletedMsgIDs(delmsgids) }()

		sp.log.LogPrintf(DEBUG, "EXECMOD %s start", pi.MessageID)

		// we should execute it
		delmsgids, _, err, _ = sp.execModCmd(
			tx, gpid, info.bid, bpid,
			modid, modCC,
			pi, tmpfns, pi.MessageID,
			cref, delmsgids, delModIDState{})
		if err != nil {
			unexpected = true
			return
		}

		sp.log.LogPrintf(DEBUG, "EXECMOD %s done", pi.MessageID)
	}

	// parse msg itself
	srefs, irefs := ibref_nntp.ParseReferences(pi.MI.Message)
	// In-Reply-To helps
	prefs :=
		mail.ExtractAllValidReferences(nil, H.GetFirst("In-Reply-To"))
	// do processing
	err = sp.processRefsAfterPost(
		tx,
		srefs, irefs, prefs,
		info.bid, info.tid, bpid,
		pi.ID, info.Newsgroup, pi.MessageID)
	if err != nil {
		unexpected = true
		return
	}

	// move files
	sp.log.LogPrint(DEBUG, "moving form temporary files to their intended place")

	srcdir := sp.src.Main()
	thmdir := sp.thm.Main()

	for x := range tmpfns {
		from := tmpfns[x]
		to := srcdir + pi.FI[x].ID
		sp.log.LogPrintf(DEBUG, "renaming %q -> %q", from, to)
		xe := fu.RenameNoClobber(from, to)
		if xe != nil {
			if os.IsExist(xe) {
				//sp.log.LogPrintf(DEBUG, "failed to rename %q to %q: %v", from, to, xe)
			} else {
				err = fmt.Errorf("failed to rename %q to %q: %v", from, to, xe)
				sp.log.LogPrint(ERROR, err.Error())
				unexpected = true
				return
			}
			os.Remove(from)
		}
	}

	for x := range tmpthmfns {
		from := tmpthmfns[x]
		if from == "" {
			continue
		}
		to := thmdir + pi.FI[x].Thumb

		sp.log.LogPrintf(DEBUG, "thm renaming %q -> %q", from, to)

		xe := fu.RenameNoClobber(from, to)
		if xe != nil {
			if os.IsExist(xe) {
				//sp.log.LogPrintf(DEBUG, "failed to rename %q to %q: %v", from, to, xe)
			} else {
				err = fmt.Errorf("failed to rename %q to %q: %v", from, to, xe)
				sp.log.LogPrint(ERROR, err.Error())
				unexpected = true
				return
			}
			os.Remove(from)
		}
	}

	// commit
	sp.log.LogPrintf(DEBUG, "nntppost commit start")
	err = tx.Commit()
	if err != nil {
		err = sp.sqlError("nntp tx commit", err)
		unexpected = true
		return
	}
	sp.log.LogPrintf(DEBUG, "nntppost commit done")

	return
}
