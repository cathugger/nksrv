package psqlib

import (
	"fmt"
	"io"
	"mime"
	nmail "net/mail"
	"os"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/date"
	. "nekochan/lib/logx"
	"nekochan/lib/mail"
	"nekochan/lib/mailib"
	"nekochan/lib/nntp"
)

// mandatory headers for transmission. POST uses separate system
var hdrNNTPMandatory = [...]struct {
	h string // header
	o bool   // optional (allow absence?)
}{
	// NetNews stuff specified in {RFC 5536}
	{"Message-ID", true}, // special handling
	{"From", false},
	{"Date", false},
	{"Newsgroups", false},
	{"Path", true},    // more lax than {RFC 5536}
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
}

type nntpParsedInfo struct {
	insertSqlInfo
	mailib.ParsedMessageInfo
}

func (sp *PSQLIB) nntpDigestTransferHead(
	w Responder, H mail.Headers, unsafe_sid CoreMsgIDStr) (
	info nntpParsedInfo, err error, unexpected bool) {

	for _, mv := range hdrNNTPMandatory {
		hv := H[mv.h]
		if !mv.o && len(hv) != 1 {
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
		H["Content-Type"] = H["Content-Type"][:1]
		H["Content-Type"][0].V = au.TrimWSString(H["Content-Type"][0].V)
	}
	if len(H["Content-Transfer-Encoding"]) != 0 {
		H["Content-Transfer-Encoding"] = H["Content-Transfer-Encoding"][:1]
		H["Content-Transfer-Encoding"][0].V =
			au.TrimWSString(H["Content-Transfer-Encoding"][0].V)
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

		if unsafe_sid != cid {
			err = fmt.Errorf(
				"IHAVE Message-ID <%s> doesn't match article Message-ID <%s>",
				unsafe_sid, cid)
			return
		}

		info.MessageID = cid

	} else {
		fmsgids := fmt.Sprintf("<%s>", unsafe_sid)
		H["Message-ID"] = mail.OneHeaderVal(fmsgids)
		info.MessageID = cutMsgID(FullMsgIDStr(fmsgids))
	}

	// Date
	pdate, err := mail.ParseDate(H["Date"][0].V)
	if err != nil {
		err = fmt.Errorf("error parsing Date header: %v", err)
		return
	}
	info.PostedDate = pdate.Unix()
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
		}
		return
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

func isSubjectEmpty(s string) bool {
	return s == "" || au.EqualFoldString(s, "None") ||
		au.EqualFoldString(s, "no subject") ||
		au.StartsWithFoldString(s, "Re: ")
	/*
	 * XXX tbh unsure about "Re: " but to precisely check that,
	 * would need to peek into post it refers to
	 */
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
	name string, H mail.Headers, info nntpParsedInfo) {

	defer os.Remove(name)

	f, err := os.Open(name)
	if err != nil {
		sp.log.LogPrintf(WARN,
			"nntpProcessArticle: failed to open: %v", err)
		return
	}
	defer f.Close()

	// TODO skip headers because we already have them
	mh, err := mail.ReadHeaders(f, 2<<20)
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

	pi.Date = date.UnixTimeUTC(info.PostedDate)

	if len(H["Subject"]) != 0 {
		sh := H["Subject"][0].V
		ssub := au.TrimWSString(sh)
		if !isSubjectEmpty(ssub) {
			if len(H["MIME-Version"]) != 0 {
				// undo MIME hacks, if any
				dsub, e := mimeWordDecoder.Decode(ssub)
				if e == nil {
					ssub = dsub
				}
			}
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

	// TODO
	sp.log.LogPrintf(DEBUG, "nntpProcessArticle: pi: %#v", pi)
	sp.log.LogPrintf(DEBUG, "nntpProcessArticle: tfns: %#v", tfns)

	for _, fn := range tfns {
		os.Remove(fn)
	}
}
