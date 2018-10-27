package psqlib

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	qp "mime/quotedprintable"
	"strings"
	"unicode/utf8"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/mail"
	"nekochan/lib/nntp"

	"golang.org/x/text/encoding/ianaindex"
)

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

func optimalTextMessageString(s string) (x string) {
	x = strings.Replace(s, "\r", "", -1)
	// last \n isn't needed if char before it exists and it isn't \n
	if len(x) > 1 && x[len(x)-1] == '\n' && x[len(x)-2] != '\n' {
		x = x[:len(x)-1]
	}
	return
}

func (sp *PSQLIB) devourTransferArticle(
	XH mail.Headers, info nntpParsedInfo, xr io.Reader) (
	pi postInfo, err error) {

	ismultipart := strings.HasPrefix(info.ContentType, "multipart/")

	var cte string
	if len(XH["Content-Transfer-Encoding"]) != 0 {
		cte = XH["Content-Transfer-Encoding"][0]
	}

	var xbinary bool

	if cte == "" ||
		au.EqualFoldString(cte, "7bit") ||
		au.EqualFoldString(cte, "8bit") {

		xbinary = false
	} else if au.EqualFoldString(cte, "base64") {
		if ismultipart {
			err = errors.New("multipart x base64 not allowed")
			return
		}
		xr = base64.NewDecoder(base64.StdEncoding, xr)
		xbinary = true
	} else if au.EqualFoldString(cte, "quoted-printable") {
		if ismultipart {
			err = errors.New("multipart x quoted-printable not allowed")
			return
		}
		xr = qp.NewReader(xr)
		xbinary = false
	} else if au.EqualFoldString(cte, "binary") {
		xbinary = true
	} else {
		err = fmt.Errorf("unknown Content-Transfer-Encoding: %s", cte)
		return
	}

	// we won't need this anymore
	delete(XH, "Content-Transfer-Encoding")

	textprocessed := false

	guttleBody := func(
		r io.Reader, H mail.Headers, ct string, cpar map[string]string,
		binary bool) (obj bodyObject, err error) {

		var n int64

		// is used when message is properly decoded
		hideattachment := false

		if !textprocessed && (ct == "" || strings.HasPrefix(ct, "text/")) {
			// try processing as main text
			// even if we fail, don't try doing it with other part
			textprocessed = true

			// TODO make configurable
			const defaultTextBuf = 512
			const maxTextBuf = (32 << 10) - 1

			b := &strings.Builder{}
			b.Grow(defaultTextBuf)

			n, err = io.CopyN(b, r, maxTextBuf+1)
			if err != nil {
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
						pi.MI.Message = str
						obj.Data = postObjectIndex(0)
						return

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
							dstr = normalizeTextMessage(dstr)
							pi.MI.Message = dstr
							hideattachment = true
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
		}

		// if this point is reached, we'll need to add this as attachment

		f, err := sp.src.TempFile("nntp-", "")
		if err != nil {
			return
		}
		n, err = io.Copy(f, r)
		if err != nil {
			return
		}
		err = f.Close()
		if err != nil {
			return
		}
		panic("TODO")
		_ = hideattachment
		return

	}

	if !ismultipart {
		pi.L.Body, err =
			guttleBody(xr, XH, info.ContentType, info.ContentParams, xbinary)
		pi.L.Binary = xbinary
		panic("TODO")
	} else {
		// we're not going to save parameters of this
		XH["Content-Type"][0] =
			au.TrimWSString(au.UntilString(XH["Content-Type"][0], ';'))
		// TODO
		panic("TODO")
	}
}
