package psqlib

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/ed25519"

	"nksrv/lib/date"
	"nksrv/lib/ibref_nntp"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mailib"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
)

// expensive processing after initial DB lookup but before commit
func (sp *PSQLIB) wp_act_process(ctx *wp_context) (err error) {
	// use normalised forms
	// theorically, normalisation could increase size sometimes, which could lead to rejection of previously-fitting message
	// but it's better than accepting too big message, as that could lead to bad things later on
	ctx.pInfo.MI.Title = strings.TrimSpace(optimiseFormLine(ctx.xf.title))
	ctx.pInfo.MI.Author = strings.TrimSpace(optimiseFormLine(ctx.xf.name))

	var signkeyseed []byte
	if i := strings.IndexByte(ctx.pInfo.MI.Author, '#'); i >= 0 {
		tripstr := ctx.pInfo.MI.Author[i+1:]
		// strip stuff to not leak secrets
		ctx.pInfo.MI.Author = strings.TrimSpace(ctx.pInfo.MI.Author[:i])

		// we currently only support ed25519 seed syntax
		tripseed, e := hex.DecodeString(tripstr)
		if e != nil || len(tripseed) != ed25519.SeedSize {
			err = badWebRequest(errInvalidTripcode)
			return
		}
		signkeyseed = tripseed
	}

	ctx.pInfo.MI.Message = tu.NormalizeTextMessage(ctx.xf.message)
	ctx.pInfo.MI.Sage = ctx.isReply &&
		(ctx.postOpts.sage || strings.ToLower(ctx.pInfo.MI.Title) == "sage")

	// check for specified limits
	var filecount int
	err, filecount = checkSubmissionLimits(&ctx.postLimits, ctx.isReply, ctx.f, ctx.pInfo.MI)
	if err != nil {
		err = badWebRequest(err)
		return
	}

	// disallow content-less msgs
	if len(ctx.pInfo.MI.Message) == 0 &&
		filecount == 0 &&
		(len(signkeyseed) == 0 || len(ctx.pInfo.MI.Title) == 0) {

		err = badWebRequest(errEmptyMsg)
		return
	}

	// time awareness
	tu := date.NowTimeUnix()
	// yeah we intentionally strip nanosec part
	ctx.pInfo.Date = date.UnixTimeUTC(tu)
	// could happen if OP' time is too far into the future
	// or our time too far into the past
	// result would be invalid so disallow
	if ctx.isReply && ctx.pInfo.Date.Before(ctx.opdate.Time) {
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

	tplan := sp.pickThumbPlan(ctx.isReply, ctx.pInfo.MI.Sage)

	// process files
	ctx.pInfo.FI = make([]mailib.FileInfo, filecount)
	x := 0
	sp.log.LogPrint(DEBUG, "processing form files")
	for _, fieldname := range FileFields {
		files := ctx.f.Files[fieldname]
		for i := range files {
			ctx.pInfo.FI[x].Original = files[i].FileName
			ctx.pInfo.FI[x].Size = files[i].Size

			var ext string
			ctx.pInfo.FI[x], ext, err = generateFileConfig(
				files[i].F, files[i].ContentType, ctx.pInfo.FI[x])
			if err != nil {
				return
			}

			// thumbnail and close file
			var res thumbnailer.ThumbResult
			res, err = sp.thumbnailer.ThumbProcess(
				files[i].F,
				ext, pInfo.FI[x].ContentType, files[i].Size,
				tplan.ThumbConfig)
			if err != nil {
				err = fmt.Errorf("error thumbnailing file: %v", err)
				return
			}

			ctx.pInfo.FI[x].Type = res.FI.Kind
			if res.FI.DetectedType != "" {
				ctx.pInfo.FI[x].ContentType = res.FI.DetectedType
				// XXX change
			}
			// save it
			ctx.pInfo.FI[x].Extras.ContentType = pInfo.FI[x].ContentType
			// thumbnail
			if res.DBSuffix != "" {
				pInfo.FI[x].ThumbField = tplan.Name + "." + res.DBSuffix
				pInfo.FI[x].ThumbAttrib.Width = uint32(res.Width)
				pInfo.FI[x].ThumbAttrib.Height = uint32(res.Height)

				dname := pInfo.FI[x].ID + "." + tplan.Name + "." + res.CF.Suffix
				ctx.thumbMoves = append(ctx.thumbMoves, thumbMove{
					FullTmpName: res.CF.FullTmpName,
					RelDestName: dname,
				})
				for _, ce := range res.CE {
					dname = pInfo.FI[x].ID + "." + tplan.Name + "." + ce.Suffix
					ctx.thumbMoves = append(ctx.thumbMoves, thumbMove{
						FullTmpName: ce.FullTmpName,
						RelDestName: dname,
					})
				}
			}
			if len(res.FI.Attrib) != 0 {
				pInfo.FI[x].FileAttrib = res.FI.Attrib
			}

			for xx := 0; xx < x; xx++ {
				if ctx.pInfo.FI[xx].Equivalent(ctx.pInfo.FI[x]) {
					err = badWebRequest(errDuplicateFile(xx, x))
					return
				}
			}

			x++
		}
	}

	// is control message?
	ctx.isctlgrp = board == "ctl"

	// process references
	ctx.srefs, ctx.irefs = ibref_nntp.ParseReferences(ctx.pInfo.MI.Message)
	var inreplyto []string
	// we need to build In-Reply-To beforehand
	// best-effort basis, in most cases it'll be okay
	inreplyto, err = sp.processReferencesOnPost(
		sp.db.DB, ctx.srefs, ctx.bid, postID(ctx.tid.Int64), ctx.isctlgrp)
	if err != nil {
		return
	}

	// fill in layout/sign
	var fmsgids FullMsgIDStr
	var fref FullMsgIDStr
	cref := CoreMsgIDStr(ctx.ref.String)
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
}
