package pipostnntp

import (
	"fmt"
	"io"
	"unicode/utf8"

	"nksrv/lib/app/base/mailibsign"
	"nksrv/lib/app/mailib"
	"nksrv/lib/mail"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/utils/date"
	. "nksrv/lib/utils/logx"
	au "nksrv/lib/utils/text/asciiutils"
	tu "nksrv/lib/utils/text/textutils"
)

func isInnerMessage(t string, h mail.HeaderMap) bool {
	return (t == "message/rfc822" || t == "message/global") &&
		len(h["Content-Disposition"]) == 0
}

func (ctx *postNNTPContext) pn_eatbody(
	br io.Reader) (err error, unexpected bool) {

	ctx.isSage = ctx.info.isReply && len(ctx.H["X-Sage"]) != 0

	tplan := ctx.sp.pickThumbPlan(ctx.info.isReply, ctx.isSage)
	texec := thumbnailer.ThumbExec{
		Thumbnailer: ctx.sp.thumbnailer,
		ThumbPlan:   tplan,
	}

	act_t, act_par := mailib.ProcessContentType(ctx.H.GetFirst("Content-Type"))
	eatinner := isInnerMessage(act_t, ctx.H)

	ver, iow := mailibsign.PrepareVerifier(ctx.H, act_t, act_par, eatinner)

	var IH mail.HeaderMap
	ctx.pi, ctx.tmpfns, ctx.thumbInfos, IH, err =
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
		ctx.pi.MI.Trip = res.PubKey
	}
	verifiedinner := pubkeystr != ""

	// properly fill in fields

	if ctx.info.FullMsgIDStr == "" {
		// was POST, think of Message-ID there
		fmsgids := mailib.NewRandomMessageID(ctx.info.PostedDate, ctx.sp.instance)
		ctx.H["Message-ID"] = mail.OneHeaderVal(string(fmsgids))
		ctx.info.FullMsgIDStr = fmsgids
	}

	ctx.pi.H = ctx.H
	ctx.pi.MessageID = cutMsgID(ctx.info.FullMsgIDStr)
	ctx.pi.ID = mailib.HashPostID_SHA1(ctx.info.FullMsgIDStr)
	ctx.pi.Date = date.UnixTimeUTC(ctx.info.PostedDate)

	ctx.pi.FC = countRealFiles(ctx.pi.FI)

	if ver != nil {
		if verifiedinner {
			ctx.sp.log.LogPrintf(
				DEBUG, "sigver: %s successfuly verified as %s",
				ctx.info.FullMsgIDStr, ctx.pi.MI.Trip)
		} else {
			ctx.sp.log.LogPrintf(
				DEBUG, "sigver: %s failed verification",
				ctx.info.FullMsgIDStr)
		}
	}

	if IH != nil && verifiedinner {
		// validated inner msg, should take Subject and other hdrs from it
		ctx.H = IH
	}

	if ctx.H.Has("Subject") {
		sh := ctx.H["Subject"][0].V

		ssub := sh

		if ctx.H.Has("MIME-Version") {
			// undo MIME hacks, if any
			dsub, e := mail.DecodeMIMEWordHeader(ssub)
			if e == nil {
				ssub = dsub
			}
		}

		// ensure safety and sanity
		ssub = au.TrimWSString(safeHeader(tu.TruncateText(ssub, maxSubjectSize)))

		if !isSubjectEmpty(ssub, ctx.info.isReply, ctx.isSage, ctx.info.refSubject) {
			ctx.pi.MI.Title = ssub
			if ctx.pi.MI.Title == sh && len(ctx.H["Subject"]) == 1 {
				// no need to duplicate
				delete(ctx.H, "Subject")
			}
		}
	}

	if fromhdr := au.TrimWSString(ctx.H.GetFirst("From")); fromhdr != "" {

		a, e := mail.ParseAddressX(fromhdr)
		if e == nil && utf8.ValidString(a.Name) {
			// XXX should we filter out "Anonymous" names? would save some bytes
			ctx.pi.MI.Author = au.TrimWSString(safeHeader(
				tu.TruncateText(a.Name, maxNameSize)))
		} else {
			ctx.pi.MI.Author = "[Invalid From header]"
		}
	}

	ctx.pi.MI.Sage = ctx.isSage

	return
}
