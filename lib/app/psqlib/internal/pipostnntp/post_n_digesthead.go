package pipostnntp

import (
	"errors"
	"fmt"
	"time"

	"nksrv/lib/app/mailib"
	"nksrv/lib/app/psqlib/internal/pibase"
	au "nksrv/lib/utils/text/asciiutils"
	"nksrv/lib/mail"
	"nksrv/lib/nntp"
	"nksrv/lib/utils/date"
)

// extracts info from main message headers into structure
func nntpDigestTransferHead(
	sp *pibase.PSQLIB,
	H mail.HeaderMap, unsafe_sid TCoreMsgIDStr, expectgroup string,
	post, notrace bool) (
	info nntpParsedInfo, err error, unexpected bool,
	wantroot TFullMsgIDStr) {

	var restrictions []headerRestriction
	if !post {
		restrictions = hdrNNTPTransferRestrict[:]
	} else {
		restrictions = hdrNNTPPostRestrict[:]
	}
	for _, mv := range restrictions {
		hv := H[mv.h]
		if !mv.o && (len(hv) != 1 || au.TrimWSString(hv[0].V) == "") {
			err = fmt.Errorf("exactly one %q header required", mv.h)
			return
		}
		if mv.o && len(hv) > 1 {
			err = fmt.Errorf("more than one %q header not allowed", mv.h)
			return
		}
	}

	// delete garbage
	delete(H, "Relay-Version")
	delete(H, "Date-Received")
	delete(H, "Xref")

	mailib.CleanContentTypeAndTransferEncoding(H)

	// Message-ID validation
	hmsgid := H.GetOneOrNone("Message-ID")

	if hmsgid != "" {

		// this should be done at parsing time already
		// yes we modify header there
		// hmsgids[0].V = au.TrimWSString(hmsgids[0].V)

		hid := TFullMsgIDStr(hmsgid)

		if !validMsgID(hid) {
			err = fmt.Errorf("invalid article Message-ID %q", hid)
			return
		}

		cid := cutMsgID(hid)

		if unsafe_sid != cid && unsafe_sid != "" {
			// check for mismatch of provided
			err = fmt.Errorf(
				"provided Message-ID <%s> doesn't match article Message-ID <%s>",
				unsafe_sid, cid)
			return
		}

		info.FullMsgIDStr = hid

	} else if unsafe_sid != "" {
		// no Message-ID header but we have Message-ID given to us via other means
		// just make it on our own
		fmsgids := fmt.Sprintf("<%s>", unsafe_sid)
		H["Message-ID"] = mail.OneHeaderVal(fmsgids)
		info.FullMsgIDStr = TFullMsgIDStr(fmsgids)
	} else if !post {
		// no known Message-ID and not POST, error out
		err = errors.New("missing Message-ID")
		return
	}
	// incase POST and no Message-ID, we'll figure it out later

	// Newsgroups
	hgroup := au.TrimWSString(H["Newsgroups"][0].V)
	// normally allowed multiple ones, separated by `,` and space,
	// but we only support single-board posts
	if !nntp.ValidGroupSlice(unsafeStrToBytes(hgroup)) {
		err = fmt.Errorf("newsgroup %q not supported", hgroup)
		return
	}
	if expectgroup != "" && hgroup != expectgroup {
		err = fmt.Errorf("newsgroup %q not expected", hgroup)
		return
	}
	info.Newsgroup = hgroup

	// References
	if len(H["References"]) != 0 {
		info.FRef = mail.ExtractFirstValidReference(H["References"][0].V)
		if info.FRef == info.FullMsgIDStr && info.FullMsgIDStr != "" {
			if post {
				err = errors.New("self-references not allowed")
				return
			}
			info.FRef = ""
		}
	}

	// Date
	nowtimeu := date.NowTimeUnix()
	if len(H["Date"]) != 0 {
		hdate := H.GetFirst("Date")
		// NOTE: incase POST we try to parse in more strict way
		// limiting syntax of non-POST stuff would hurt propagation
		pdate, e := mail.ParseDateX(hdate, !post)
		if e != nil {
			err = fmt.Errorf("error parsing Date header %q: %v", hdate, e)
			return
		}
		info.PostedDate = pdate.Unix()
		// check if message is not too new
		const maxFutureSecs = 15 * 60 // 15 minutes
		if info.PostedDate-maxFutureSecs > nowtimeu {
			// XXX soft error, have a way to serialize it into condition
			err = errors.New("date is too far in the future")
			return
		}
		// TODO check for too old aswell
		// checking for too old may help to clean up message reject/ban filters
		// however I'm not sure if we can reliably obtain cutoff
		// TODO think of & document cutoff handling
	} else {
		// incase POST and has no Date field, make one
		H["Date"] = mail.OneHeaderVal(mail.FormatDate(time.Unix(nowtimeu, 0)))
		info.PostedDate = nowtimeu
	}

	// actual DB check on group and refered article
	var wr bool
	info.insertSqlInfo, err, unexpected, wr =
		acceptArticleHead(sp, hgroup, info.FRef, info.PostedDate)
	if err != nil {
		if err == pibase.ErrNoSuchBoard {
			err = fmt.Errorf("newsgroup %q not wanted", hgroup)
		} else if err == pibase.ErrNoSuchThread {
			err = fmt.Errorf(
				"refering to non-existing root post %s not allowed", info.FRef)
		}
		if wr {
			wantroot = info.FRef
		}
		return
	}

	if len(H["Path"]) != 0 && au.TrimWSString(H["Path"][0].V) != "" {
		if !notrace {
			H["Path"][0].V = sp.Instance + "!" + H["Path"][0].V
		}
	} else {
		H["Path"] = mail.OneHeaderVal(sp.Instance + "!.POSTED!not-for-mail")
	}

	//sp.log.LogPrintf(DEBUG, "nntpDigestTransferHead done")

	return
}
