package psqlib

import (
	"errors"
	"fmt"
	"io"
	"mime"
	nmail "net/mail"
	"os"
	"time"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/date"
	fu "nekochan/lib/fileutil"
	. "nekochan/lib/logx"
	"nekochan/lib/mail"
	"nekochan/lib/mailib"
	"nekochan/lib/nntp"
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
	{"From", false},
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
	H mail.Headers, unsafe_sid CoreMsgIDStr, post bool) (
	info nntpParsedInfo, err error, unexpected bool) {

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

	// ignore other headers than first
	if len(H["Content-Type"]) != 0 {
		ts := au.TrimWSString(H["Content-Type"][0].V)
		if ts != "" {
			H["Content-Type"] = H["Content-Type"][:1]
			H["Content-Type"][0].V = ts
		} else {
			delete(H, "Content-Type")
		}
	}
	if len(H["Content-Transfer-Encoding"]) != 0 {
		ts := au.TrimWSString(H["Content-Transfer-Encoding"][0].V)
		if ts != "" {
			H["Content-Transfer-Encoding"] = H["Content-Transfer-Encoding"][:1]
			H["Content-Transfer-Encoding"][0].V = ts
		} else {
			delete(H, "Content-Transfer-Encoding")
		}
	}

	var tu int64
	if post {
		tu = date.NowTimeUnix()
	}

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
				"IHAVE Message-ID <%s> doesn't match article Message-ID <%s>",
				unsafe_sid, cid)
			return
		}

		info.FullMsgIDStr = hid

	} else if unsafe_sid != "" {
		fmsgids := fmt.Sprintf("<%s>", unsafe_sid)
		H["Message-ID"] = mail.OneHeaderVal(fmsgids)
		info.FullMsgIDStr = FullMsgIDStr(fmsgids)
	} else if post {
		fmsgids := mailib.NewRandomMessageID(tu, sp.instance)
		H["Message-ID"] = mail.OneHeaderVal(string(fmsgids))
		info.FullMsgIDStr = fmsgids
	} else {
		err = errors.New("missing Message-ID")
		return
	}

	// Date
	if len(H["Date"]) != 0 {
		pdate, e := mail.ParseDate(H["Date"][0].V)
		if e != nil {
			err = fmt.Errorf("error parsing Date header: %v", e)
			return
		}
		info.PostedDate = pdate.Unix()
	} else {
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
	info.Newsgroup = hgroup

	// References
	var troot FullMsgIDStr
	if len(H["References"]) != 0 {
		troot = mail.ExtractFirstValidReference(H["References"][0].V)
	}

	// actual DB check on group and refered article
	info.insertSqlInfo, err, unexpected = sp.acceptArticleHead(hgroup, troot)
	if err != nil {
		if err == errNoSuchBoard {
			err = fmt.Errorf("newsgroup %q not wanted", hgroup)
		} else if err == errNoSuchThread {
			err = fmt.Errorf(
				"refering to non-existing root post %s not allowed", troot)
		}
		return
	}

	if len(H["Path"]) != 0 && au.TrimWSString(H["Path"][0].V) != "" {
		H["Path"][0].V = sp.instance + "!" + H["Path"][0].V
	} else {
		H["Path"] = mail.OneHeaderVal(sp.instance + "!.POSTED!not-for-mail")
	}

	// Content-Type
	if len(H["Content-Type"]) != 0 {
		var e error
		info.ContentType, info.ContentParams, e =
			mime.ParseMediaType(H["Content-Type"][0].V)
		if e != nil {
			info.ContentType = "invalid"
		}
	}

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

type failCharsetError string

func (f failCharsetError) Error() string {
	return fmt.Sprintf("unknown charset: %q", string(f))
}

func failCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	return nil, failCharsetError(charset)
}

var mimeWordDecoder = mime.WordDecoder{CharsetReader: failCharsetReader}

func (sp *PSQLIB) nntpProcessArticle(
	r io.Reader, H mail.Headers, info nntpParsedInfo) {

	// TODO skip headers because we already have them
	mh, err := mail.ReadHeaders(r, 2<<20)
	if err != nil {
		sp.log.LogPrintf(WARN,
			"nntpProcessArticle: failed reading headers: %v", err)
		return
	}
	defer mh.Close()

	pi, tfns, err := mailib.DevourMessageBody(
		&sp.src, H, info.ParsedMessageInfo, mh.B)
	if err != nil {
		sp.log.LogPrintf(WARN,
			"nntpProcessArticle: devourTransferArticle failed: %v", err)
		return
	}

	if len(pi.FI) != len(tfns) {
		panic("len(pi.FI) != len(tfns)")
	}

	// properly fill in fields

	pi.MessageID = cutMsgID(info.FullMsgIDStr)
	pi.ID = mailib.HashPostID_SHA1(info.FullMsgIDStr)
	pi.Date = date.UnixTimeUTC(info.PostedDate)

	if len(H["Subject"]) != 0 {
		sh := H["Subject"][0].V
		ssub := au.TrimWSString(sh)
		if len(H["MIME-Version"]) != 0 {
			// undo MIME hacks, if any
			dsub, e := mimeWordDecoder.Decode(ssub)
			if e == nil {
				ssub = dsub
			}
		}
		if !isSubjectEmpty(ssub, info.isReply, info.refSubject) {
			pi.MI.Title = ssub
			if pi.MI.Title == sh && len(H["Subject"]) == 1 {
				// no need to duplicate
				delete(H, "Subject")
			}
		}
	}

	if len(H["From"]) != 0 {
		a, e := nmail.ParseAddress(H["From"][0].V)
		if e == nil {
			// XXX should we filter out "Anonymous" names? would save some bytes
			pi.MI.Author = a.Name
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
		sp.log.LogPrintf(
			ERROR, "nntpProcessArticle: post insertion failed: %v", err)

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
				sp.log.LogPrintf(DEBUG, "failed to rename %q to %q: %v", from, to, xe)
			} else {
				sp.log.LogPrintf(ERROR, "failed to rename %q to %q: %v", from, to, xe)
			}
			os.Remove(from)
		}
	}
}
