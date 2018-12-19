package psqlib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	au "centpd/lib/asciiutils"
	"centpd/lib/date"
	fu "centpd/lib/fileutil"
	. "centpd/lib/logx"
	"centpd/lib/mail"
	"centpd/lib/mailib"
	"centpd/lib/nntp"
)

type headerRestriction struct {
	h string // header
	o bool   // optional (allow absence?)
}

var hdrNNTPPostRestrict = [...]headerRestriction{
	// NetNews stuff specified in {RFC 5536}
	{"Message-ID", true},
	{"From", false},
	{"Date", true},
	{"Newsgroups", false},
	{"Path", true},
	{"Subject", true}, // more lax than {RFC 5536} (no subject is much better than "none")

	// {RFC 5322}
	{"Sender", true},
	{"Reply-To", true},
	{"To", true},
	{"Cc", true},
	{"Bcc", true},
	{"In-Reply-To", true},
	{"References", true},

	// some extras we process
	{"Injection-Date", true},
	{"NNTP-Posting-Date", true},
}

// mandatory headers for transmission. POST uses separate system
var hdrNNTPTransferRestrict = [...]headerRestriction{
	// NetNews stuff specified in {RFC 5536}
	{"Message-ID", true}, // special handling
	{"From", true},       // idfk why there are articles like this
	{"Date", false},
	{"Newsgroups", false},
	{"Path", false},
	{"Subject", true}, // more lax than {RFC 5536} (no subject is much better than "none")

	// {RFC 5322}
	{"Sender", true},
	{"Reply-To", true},
	{"To", true},
	{"Cc", true},
	{"Bcc", true},
	{"In-Reply-To", true},
	{"References", true},

	// some extras we process
	{"Injection-Date", true},
	{"NNTP-Posting-Date", true},
}

type insertSqlInfo struct {
	postLimits submissionLimits
	threadOpts threadOptions
	tid        postID
	bid        boardID
	isReply    bool
	refSubject string
}

type nntpParsedInfo struct {
	insertSqlInfo
	mailib.ParsedMessageInfo
}

func (sp *PSQLIB) nntpDigestTransferHead(
	H mail.Headers, unsafe_sid CoreMsgIDStr, expectgroup string, post bool) (
	info nntpParsedInfo, err error, unexpected bool, wantroot FullMsgIDStr) {

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
	hmsgids := H["Message-ID"]
	if len(hmsgids) != 0 {

		// yes we modify header there
		hmsgids[0].V = au.TrimWSString(hmsgids[0].V)

		hid := FullMsgIDStr(hmsgids[0].V)

		if !validMsgID(hid) {
			err = fmt.Errorf("invalid article Message-ID %q", hid)
			return
		}

		cid := cutMsgID(hid)

		if unsafe_sid != cid && unsafe_sid != "" {
			err = fmt.Errorf(
				"provided Message-ID <%s> doesn't match article Message-ID <%s>",
				unsafe_sid, cid)
			return
		}

		info.FullMsgIDStr = hid

	} else if unsafe_sid != "" {
		fmsgids := fmt.Sprintf("<%s>", unsafe_sid)
		H["Message-ID"] = mail.OneHeaderVal(fmsgids)
		info.FullMsgIDStr = FullMsgIDStr(fmsgids)
	} else if !post {
		err = errors.New("missing Message-ID")
		return
	}

	// Date
	if len(H["Date"]) != 0 {
		hdate := H.GetFirst("Date")
		pdate, e := mail.ParseDateX(hdate)
		if e != nil {
			err = fmt.Errorf("error parsing Date header %q: %v", hdate, e)
			return
		}
		info.PostedDate = pdate.Unix()
	} else {
		tu := date.NowTimeUnix()
		H["Date"] = mail.OneHeaderVal(mail.FormatDate(time.Unix(tu, 0)))
		info.PostedDate = tu
	}

	// TODO check if message is not too new
	// maybe check for too old aswell
	// checking for too old may help to clean up message reject/ban filters

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
	var troot FullMsgIDStr
	if len(H["References"]) != 0 {
		troot = mail.ExtractFirstValidReference(H["References"][0].V)
		if troot == info.FullMsgIDStr && info.FullMsgIDStr != "" {
			if post {
				err = errors.New("self-references not allowed")
				return
			}
			troot = ""
		}
	}

	// actual DB check on group and refered article
	var wr bool
	info.insertSqlInfo, err, unexpected, wr =
		sp.acceptArticleHead(hgroup, troot, info.PostedDate)
	if err != nil {
		if err == errNoSuchBoard {
			err = fmt.Errorf("newsgroup %q not wanted", hgroup)
		} else if err == errNoSuchThread {
			err = fmt.Errorf(
				"refering to non-existing root post %s not allowed", troot)
		}
		if wr {
			wantroot = troot
		}
		return
	}

	if len(H["Path"]) != 0 && au.TrimWSString(H["Path"][0].V) != "" {
		H["Path"][0].V = sp.instance + "!" + H["Path"][0].V
	} else {
		H["Path"] = mail.OneHeaderVal(sp.instance + "!.POSTED!not-for-mail")
	}

	//sp.log.LogPrintf(DEBUG, "nntpDigestTransferHead done")

	return
}

func isSubjectEmpty(s string, isReply bool, refs string) bool {
	isVoid := func(x string) bool {
		// content-less subjects some shitty nodes like spamming
		return x == "" || au.EqualFoldString(x, "None") ||
			au.EqualFoldString(x, "no subject")
	}
	if isVoid(s) {
		return true
	}
	if isReply {
		// content-less copy of parent subject
		if au.EqualFoldString(s, refs) {
			return true
		}
		// if after above checks it doesn't start with Re: it's legit
		if !au.StartsWithFoldString(s, "Re:") {
			return false
		}
		if len(s) > 3 && (s[3] == ' ' || s[3] == '\t') {
			s = s[4:]
		} else {
			s = s[3:]
		}
		if refs == "" {
			// parent probably was void, so check for that
			return isVoid(s)
		} else {
			// Re: parent
			return au.EqualFoldString(s, refs)
		}
	} else {
		return false
	}
}

func (sp *PSQLIB) netnewsSubmitFullArticle(
	r io.Reader, H mail.Headers, info nntpParsedInfo) {

	mh, err := mail.SkipHeaders(r)
	if err != nil {
		sp.log.LogPrintf(WARN,
			"netnewsSubmitFullArticle: failed reading headers: %v", err)
		return
	}
	defer mh.Close()

	err, unexpected := sp.netnewsSubmitArticle(mh.B, H, info)
	if err != nil {
		if !unexpected {
			sp.log.LogPrintf(WARN, "netnewsSubmitArticle: %v", err)
		} else {
			sp.log.LogPrintf(ERROR, "netnewsSubmitArticle: %v", err)
		}
	}
}

func (sp *PSQLIB) netnewsSubmitArticle(
	br io.Reader, H mail.Headers, info nntpParsedInfo) (
	err error, unexpected bool) {

	pi, tfns, err := mailib.DevourMessageBody(
		&sp.src, H, info.ParsedMessageInfo, br)
	if err != nil {
		err = fmt.Errorf("devourTransferArticle failed: %v", err)
		return
	}

	if len(pi.FI) != len(tfns) {
		panic("len(pi.FI) != len(tfns)")
	}

	// properly fill in fields

	if info.FullMsgIDStr == "" {
		// was POST, think of Message-ID there
		fmsgids := mailib.NewRandomMessageID(info.PostedDate, sp.instance)
		H["Message-ID"] = mail.OneHeaderVal(string(fmsgids))
		info.FullMsgIDStr = fmsgids
	}

	pi.MessageID = cutMsgID(info.FullMsgIDStr)
	pi.ID = mailib.HashPostID_SHA1(info.FullMsgIDStr)
	pi.Date = date.UnixTimeUTC(info.PostedDate)

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
		ssub = au.TrimWSString(safeHeader(ssub))

		if !isSubjectEmpty(ssub, info.isReply, info.refSubject) {
			pi.MI.Title = ssub
			if pi.MI.Title == sh && len(H["Subject"]) == 1 {
				// no need to duplicate
				delete(H, "Subject")
			}
		}
	}

	fromhdr := au.TrimWSString(H.GetFirst("From"))
	if fromhdr != "" {
		a, e := mail.ParseAddressX(fromhdr)
		if e == nil && utf8.ValidString(a.Name) {
			// XXX should we filter out "Anonymous" names? would save some bytes
			pi.MI.Author = au.TrimWSString(safeHeader(a.Name))
		} else {
			pi.MI.Author = "[Invalid From header]"
		}
	}

	if len(H["X-Sage"]) != 0 {
		pi.MI.Sage = true
	}

	// perform insert
	if !info.isReply {
		sp.log.LogPrint(DEBUG, "inserting newthread post data to database")
		_, err = sp.insertNewThread(info.bid, pi)
	} else {
		sp.log.LogPrint(DEBUG, "inserting reply post data to database")
		_, err = sp.insertNewReply(replyTargetInfo{
			info.bid, info.tid, info.threadOpts.BumpLimit}, pi)
	}
	if err != nil {
		err = fmt.Errorf("post insertion failed: %v", err)
		unexpected = true

		// cleanup
		for _, fn := range tfns {
			os.Remove(fn)
		}

		return
	}

	// move files
	sp.log.LogPrint(DEBUG, "moving form temporary files to their intended place")
	for x := range tfns {
		from := tfns[x]
		to := sp.src.Main() + pi.FI[x].ID
		sp.log.LogPrintf(DEBUG, "renaming %q -> %q", from, to)
		xe := fu.RenameNoClobber(from, to)
		if xe != nil {
			if os.IsExist(xe) {
				//sp.log.LogPrintf(DEBUG, "failed to rename %q to %q: %v", from, to, xe)
			} else {
				sp.log.LogPrintf(ERROR, "failed to rename %q to %q: %v", from, to, xe)
			}
			os.Remove(from)
		}
	}

	return
}
