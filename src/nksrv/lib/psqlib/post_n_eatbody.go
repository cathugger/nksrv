package psqlib

import (
	"fmt"
	"io"
	"unicode/utf8"

	au "nksrv/lib/asciiutils"
	"nksrv/lib/date"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mailib"
	"nksrv/lib/mailibsign"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
)

func isInnerMessage(t string, h main.Headers) bool {
	return (t == "message/rfc822" || t == "message/global") &&
		len(h["Content-Disposition"]) == 0
}

func (ctx *nntpPostCtx) pn_eatbody(
	br io.Reader) (err error, unexpected bool) {

	ctx.isSage = ctx.info.isReply && len(ctx.H["X-Sage"]) != 0

	tplan := sp.pickThumbPlan(ctx.info.isReply, ctx.isSage)
	texec := thumbnailer.ThumbExec{
		Thumbnailer: sp.thumbnailer,
		ThumbPlan:   tplan,
	}

	act_t, act_par := mailib.ProcessContentType(ctx.H.GetFirst("Content-Type"))
	eatinner := isInnerMessage(act_t, ctx.H)

	ver, iow := mailibsign.PrepareVerifier(ctx.H, act_t, act_par, eatinner)

	var IH mail.Headers
	ctx.pi, ctx.tmpfns, ctx.thminfos, IH, err =
		mailib.DevourMessageBody(
			&ctx.sp.src, texec, ctx.H, act_t, act_par, eatinner, br, iow)
	if err != nil {
		err = fmt.Errorf("%q devourTransferArticle failed: %v",
			ctx.info.FullMsgIDStr, err)
		return
	}

	if len(ctx.pi.FI) != len(ctx.tmpfns) {
		panic("len(pi.FI) != len(tmpfns)")
	}

	pubkeystr := ""
	if ver != nil {
		res := ver.Verify(iow)
		pubkeystr = res.PubKey
		pi.MI.Trip = res.PubKey
	}
	verifiedinner := pubkeystr != ""

	// properly fill in fields

	if ctx.info.FullMsgIDStr == "" {
		// was POST, think of Message-ID there
		fmsgids := mailib.NewRandomMessageID(ctx.info.PostedDate, sp.instance)
		ctx.H["Message-ID"] = mail.OneHeaderVal(string(fmsgids))
		ctx.info.FullMsgIDStr = fmsgids
	}

	pi.H = ctx.H
	pi.MessageID = cutMsgID(info.FullMsgIDStr)
	pi.ID = mailib.HashPostID_SHA1(info.FullMsgIDStr)
	pi.Date = date.UnixTimeUTC(info.PostedDate)

	pi.FC = countRealFiles(pi.FI)

	if ver != nil {
		if verifiedinner {
			sp.log.LogPrintf(DEBUG, "sigver: %s successfuly verified as %s", info.FullMsgIDStr, pi.MI.Trip)
		} else {
			sp.log.LogPrintf(DEBUG, "sigver: %s failed verification", info.FullMsgIDStr)
		}
	}

	if IH != nil && verifiedinner {
		// validated inner msg, should take Subject and other hdrs from it
		H = IH
	}

	if len(H["Subject"]) != 0 {
		sh := H["Subject"][0].V

		ssub := sh

		if len(H["MIME-Version"]) != 0 {
			// undo MIME hacks, if any
			dsub, e := mail.DecodeMIMEWordHeader(ssub)
			if e == nil {
				ssub = dsub
			}
		}

		// ensure safety and sanity
		ssub = au.TrimWSString(safeHeader(tu.TruncateText(ssub, maxSubjectSize)))

		if !isSubjectEmpty(ssub, info.isReply, ctx.isSage, info.refSubject) {
			pi.MI.Title = ssub
			if pi.MI.Title == sh && len(H["Subject"]) == 1 {
				// no need to duplicate
				delete(H, "Subject")
			}
		}
	}

	if fromhdr := au.TrimWSString(H.GetFirst("From")); fromhdr != "" {

		a, e := mail.ParseAddressX(fromhdr)
		if e == nil && utf8.ValidString(a.Name) {
			// XXX should we filter out "Anonymous" names? would save some bytes
			pi.MI.Author = au.TrimWSString(safeHeader(
				tu.TruncateText(a.Name, maxNameSize)))
		} else {
			pi.MI.Author = "[Invalid From header]"
		}
	}

	pi.MI.Sage = ctx.isSage
}
