package psqlib

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	qp "mime/quotedprintable"
	"os"
	"strings"
	"unicode/utf8"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/emime"
	. "nekochan/lib/logx"
	"nekochan/lib/mail"
	"nekochan/lib/nntp"

	"golang.org/x/text/encoding/ianaindex"
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

func (sp *PSQLIB) nntpDigestTransferHead(
	w Responder, H mail.Headers, unsafe_sid CoreMsgIDStr) (
	info nntpParsedInfo, ok bool) {

	var err error

	for _, mv := range hdrNNTPMandatory {
		hv := H[mv.h]
		if !mv.o && len(hv) != 1 {
			err = fmt.Errorf("exactly one %q header required", mv.h)
			w.ResTransferRejected(err)
			return
		}
		if mv.o && len(hv) > 1 {
			err = fmt.Errorf("more than one %q header not allowed", mv.h)
			w.ResTransferRejected(err)
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
		H["Content-Type"][0] = au.TrimWSString(H["Content-Type"][0])
	}
	if len(H["Content-Transfer-Encoding"]) != 0 {
		H["Content-Transfer-Encoding"] = H["Content-Transfer-Encoding"][:1]
		H["Content-Transfer-Encoding"][0] =
			au.TrimWSString(H["Content-Transfer-Encoding"][0])
	}

	// Message-ID validation
	hmsgids := H["Message-ID"]

	if len(hmsgids) != 0 {

		hmsgids[0] = au.TrimWSString(hmsgids[0]) // yes we modify header there

		hid := FullMsgIDStr(hmsgids[0])

		if !validMsgID(hid) {
			err = fmt.Errorf("invalid article Message-ID %q", hid)
			w.ResTransferRejected(err)
			return
		}

		cid := cutMsgID(hid)

		if unsafe_sid != cid {
			err = fmt.Errorf(
				"IHAVE Message-ID <%s> doesn't match article Message-ID <%s>",
				unsafe_sid, cid)
			w.ResTransferRejected(err)
			return
		}

		info.MessageID = cid

	} else {
		fmsgids := fmt.Sprintf("<%s>", unsafe_sid)
		H["Message-ID"] = []string{fmsgids}
		info.MessageID = cutMsgID(FullMsgIDStr(fmsgids))
	}

	// Date
	hdate := H["Date"][0]
	pdate, err := mail.ParseDate(hdate)
	if err != nil {
		err = fmt.Errorf("error parsing Date header: %v", err)
		w.ResTransferRejected(err)
		return
	}
	info.PostedDate = pdate.Unix()
	// TODO check if message is not too new
	// maybe check for too old aswell
	// checking for too old may help to clean up message reject/ban filters

	// Newsgroups
	hgroup := au.TrimWSString(H["Newsgroups"][0])
	// normally allowed multiple ones, separated by `,` and space,
	// but we only support single-board posts
	if !nntp.ValidGroupSlice(unsafeStrToBytes(hgroup)) {
		err = fmt.Errorf("newsgroup %q not supported", hgroup)
		w.ResTransferRejected(err)
		return
	}
	if err = sp.acceptNewsgroupArticle(hgroup); err != nil {
		if err == errNoSuchBoard {
			err = fmt.Errorf("newsgroup %q not wanted", hgroup)
			w.ResTransferRejected(err)
		} else {
			w.ResInternalError(err)
		}
		return
	}
	info.Newsgroup = hgroup

	// Content-Type
	if len(H["Content-Type"]) != 0 {
		info.ContentType, info.ContentParams, err =
			mime.ParseMediaType(H["Content-Type"][0])
		if err != nil {
			err = fmt.Errorf("error parsing Content-Type header: %v", err)
			w.ResTransferRejected(err)
			return
		}
	}

	ok = true
	return
}

func optimalTextMessageString(s string) string {
	// not removing \r because they should be removed already
	// last \n isn't needed if char before it exists and it isn't \n
	if len(s) > 1 && s[len(s)-1] == '\n' && s[len(s)-2] != '\n' {
		s = s[:len(s)-1]
	}
	return s
}

func nntpProcessArticlePrepareReader(
	cte string, ismultipart bool, r io.Reader) (
	_ io.Reader, binary bool, err error) {

	if cte == "" ||
		au.EqualFoldString(cte, "7bit") ||
		au.EqualFoldString(cte, "8bit") {

		binary = false
	} else if au.EqualFoldString(cte, "base64") {
		if ismultipart {
			err = errors.New("multipart x base64 not allowed")
			return
		}
		r = base64.NewDecoder(base64.StdEncoding, r)
		binary = true
	} else if au.EqualFoldString(cte, "quoted-printable") {
		if ismultipart {
			err = errors.New("multipart x quoted-printable not allowed")
			return
		}
		r = qp.NewReader(r)
		binary = false
	} else if au.EqualFoldString(cte, "binary") {
		binary = true
	} else {
		err = fmt.Errorf("unknown Content-Transfer-Encoding: %s", cte)
		return
	}
	return r, binary, err
}

func (sp *PSQLIB) nntpProcessArticleText(
	r io.Reader, binary bool, ct string, cpar map[string]string) (
	_ io.Reader, rstr string, finished bool, msgattachment bool, err error) {

	// TODO make configurable
	const defaultTextBuf = 512
	const maxTextBuf = (32 << 10) - 1

	b := &strings.Builder{}
	b.Grow(defaultTextBuf)

	if !binary {
		r = au.NewUnixTextReader(r)
	}
	n, err := io.CopyN(b, r, maxTextBuf+1)
	if err != nil && err != io.EOF {
		err = fmt.Errorf("error reading body: %v", err)
		return
	}

	str := b.String()
	if n <= maxTextBuf {
		// it fit
		cset := ""
		if ct != "" {
			cset = cpar["charset"]
		}

		if strings.IndexByte(str, 0) < 0 {

			// expect UTF-8 in most cases
			if ((cset == "" ||
				au.EqualFoldString(cset, "UTF-8") ||
				au.EqualFoldString(cset, "US-ASCII")) &&
				utf8.ValidString(str)) ||
				(au.EqualFoldString(cset, "ISO-8859-1") &&
					au.Is7BitString(str)) {

				// normal processing - no need to have copy
				if !binary {
					str = optimalTextMessageString(str)
				}
				return r, str, true, false, nil

			} else if cset == "" {
				// fallback to ISO-8859-1
				cset = "ISO-8859-1"
			}
		}

		// attempt to decode
		if cset != "" &&
			!au.EqualFoldString(cset, "UTF-8") &&
			!au.EqualFoldString(cset, "US-ASCII") {

			cod, e := ianaindex.MIME.Encoding(cset)
			if e == nil {
				dec := cod.NewDecoder()
				dstr, e := dec.String(str)
				// should not result in null characters
				if e == nil && strings.IndexByte(dstr, 0) < 0 {
					// we don't care about binary mode
					// because this is just converted copy
					// so might aswell normalize and optimize it further
					rstr = normalizeTextMessage(dstr)
					msgattachment = true
					// proceed with processing as attachment
				} else {
					// ignore
				}
			} else {
				// ignore
			}
		}

		// since we've read whole string, don't chain
		r = strings.NewReader(str)

	} else {
		// can't put in message
		// proceed with attachment processing
		r = io.MultiReader(strings.NewReader(str), r)
	}

	return r, rstr, false, msgattachment, nil
}

func (sp *PSQLIB) nntpProcessArticleAttachment(
	H mail.Headers, r io.Reader, binary bool, ct string) (
	fi fileInfo, fn string, err error) {

	// new
	f, err := sp.src.TempFile("nntp-", "")
	if err != nil {
		return
	}
	// get name
	fn = f.Name()
	// copy
	if !binary {
		r = au.NewUnixTextReader(r)
	}
	n, err := io.Copy(f, r)
	if err != nil {
		f.Close()
		return
	}
	// seek to 0
	_, err = f.Seek(0, 0)
	if err != nil {
		f.Close()
		return
	}
	// hash it
	h, err := makeFileHash(f)
	if err != nil {
		f.Close()
		return
	}
	// close
	err = f.Close()
	if err != nil {
		return
	}
	// determine with what filename should we store it
	cdis := ""
	if len(H["Content-Disposition"]) != 0 {
		cdis = H["Content-Disposition"][0]
	}
	oname := ""
	if cdis != "" {
		_, params, e := mime.ParseMediaType(cdis)
		if e == nil && params["filename"] != "" {
			oname = params["filename"]
			if i := strings.LastIndexAny(oname, "/\\"); i >= 0 {
				oname = oname[i+1:]
			}
		}
	}
	ext := ""
	if oname != "" {
		if i := strings.LastIndexByte(
			oname, '.'); i >= 0 && i+1 < len(oname) {

			ext = oname[i+1:]
		}
	}
	if ct == "" {
		ct = "text/plain"
	}
	if ext == "" || !emime.MIMEIsCanonical(ext, ct) {
		// attempt finding better extension
		mexts, e := emime.MIMEExtensionsByType(ct)
		if e == nil && len(mexts) != 0 {
			ext = mexts[0] // expect first to be good enough
		}
	}
	// ohwell, at this point we should probably have something
	// even if we don't, that's okay
	var iname string
	if ext != "" {
		iname = h + "." + ext
	} else {
		iname = h
	}
	// incase we have no info about original filename, give it something
	if oname == "" {
		oname = iname
	}

	fi = fileInfo{
		ContentType: ct,
		Size:        n,
		ID:          iname,
		Original:    oname,
	}
	return
}

func (sp *PSQLIB) devourTransferArticle(
	XH mail.Headers, info nntpParsedInfo, xr io.Reader) (
	pi postInfo, tmpfilenames []string, err error) {

	defer func() {
		if err != nil {
			for _, fn := range tmpfilenames {
				os.Remove(fn)
			}
			tmpfilenames = []string(nil)
		}
	}()

	xismultipart := strings.HasPrefix(info.ContentType, "multipart/")

	var xcte string
	if len(XH["Content-Transfer-Encoding"]) != 0 {
		xcte = XH["Content-Transfer-Encoding"][0]
	}

	var xbinary bool

	xr, xbinary, err = nntpProcessArticlePrepareReader(xcte, xismultipart, xr)
	if err != nil {
		return
	}

	// we won't need this anymore
	delete(XH, "Content-Transfer-Encoding")

	textprocessed := false

	guttleBody := func(
		r io.Reader, H mail.Headers, ct string, cpar map[string]string,
		binary bool) (obj bodyObject, err error) {

		// is used when message is properly decoded
		msgattachment := false

		if !textprocessed && len(H["Content-Disposition"]) == 0 &&
			(ct == "" || strings.HasPrefix(ct, "text/")) {

			// try processing as main text
			// even if we fail, don't try doing it with other part
			textprocessed = true

			var str string
			var finished bool
			r, str, finished, msgattachment, err =
				sp.nntpProcessArticleText(r, binary, ct, cpar)
			if err != nil {
				return
			}
			pi.MI.Message = str
			if finished {
				obj.Data = postObjectIndex(0)
				return
			}
		}

		// if this point is reached, we'll need to add this as attachment

		fi, fn, err := sp.nntpProcessArticleAttachment(H, r, binary, ct)
		tmpfilenames = append(tmpfilenames, fn)
		if err != nil {
			return
		}

		if msgattachment {
			fi.Type = FTypeMsg
		}
		pi.FI = append(pi.FI, fi)

		obj.Data = postObjectIndex(len(pi.FI))
		return
	}

	if !xismultipart {
		pi.L.Body, err =
			guttleBody(xr, XH, info.ContentType, info.ContentParams, xbinary)
		pi.L.Binary = xbinary
		// since this is toplvl dont set ContentType or other headers
	} else {
		// we're not going to save parameters of this
		XH["Content-Type"][0] =
			au.TrimWSString(au.UntilString(XH["Content-Type"][0], ';'))
		// TODO
		panic("TODO")
	}

	return
}

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

	pi, tfns, err := sp.devourTransferArticle(H, info, mh.B)
	if err != nil {
		sp.log.LogPrintf(WARN,
			"nntpProcessArticle: devourTransferArticle failed: %v", err)
		return
	}

	// TODO
	sp.log.LogPrintf(DEBUG,
		"nntpProcessArticle: pi: %#v; tfns: %#v", pi, tfns)

	for _, fn := range tfns {
		os.Remove(fn)
	}
}
