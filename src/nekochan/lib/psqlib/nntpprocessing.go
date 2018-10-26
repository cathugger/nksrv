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

func devourTransferArticle(
	H mail.Headers, info nntpParsedInfo, r io.Reader) (
	pi postInfo, err error) {

	ismultipart := strings.HasPrefix(info.ContentType, "multipart/")

	var cte string
	if len(H["Content-Transfer-Encoding"]) != 0 {
		cte = H["Content-Transfer-Encoding"][0]
	}

	var binary bool

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

	// we won't need this anymore
	delete(H, "Content-Transfer-Encoding")

	_ = binary

	if !ismultipart {
		if info.ContentType == "" ||
			strings.HasPrefix(info.ContentType, "text/") {

			// try processing as main text
			// TODO make configurable
			const defaultTextBuf = 2048
			const maxTextBuf = (32 << 10) - 1
			b := &strings.Builder{}
			b.Grow(defaultTextBuf)
			var n int64
			n, err = io.CopyN(b, r, maxTextBuf+1)
			if err != nil {
				err = fmt.Errorf("error reading body: %v", err)
				return
			}
			str := b.String()
			if n <= maxTextBuf {
				// it fit
				cset := ""
				if info.ContentType != "" {
					cset = info.ContentParams["charset"]
				}
				// expect UTF-8 in most cases
				if strings.IndexByte(str, 0) < 0 &&
					(((cset == "" ||
						au.EqualFoldString(cset, "UTF-8") ||
						au.EqualFoldString(cset, "US-ASCII")) &&
						utf8.ValidString(str)) ||
						(au.EqualFoldString(cset, "ISO-8859-1") &&
							au.Is7BitString(str))) {

					// normal processing - no need to have copy
					pi.MI.Message = str
					pi.L.Body.Data = postObjectIndex(0)
					pi.L.Binary = binary
					panic("TODO")
				} else {
					// not UTF-8
					panic("TODO")
				}

				panic("TODO")
				return
			}
			// can't put in message
			// proceed with attachment processing
			r = io.MultiReader(strings.NewReader(str), r)
		}
		// attachment
		panic("TODO")
	} else {
		// we're not going to save parameters of this
		H["Content-Type"][0] =
			au.TrimWSString(au.UntilString(H["Content-Type"][0], ';'))
		// TODO
		panic("TODO")
	}
}
