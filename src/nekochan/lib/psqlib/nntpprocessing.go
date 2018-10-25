package psqlib

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	qp "mime/quotedprintable"
	"strings"

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
	H mail.Headers, info nntpParsedInfo, r io.Reader) error {

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
			return errors.New("multipart x base64 not allowed")
		}
		r = base64.NewDecoder(base64.StdEncoding, r)
		binary = true
	} else if au.EqualFoldString(cte, "quoted-printable") {
		if ismultipart {
			return errors.New("multipart x quoted-printable not allowed")
		}
		r = qp.NewReader(r)
		binary = false
	} else if au.EqualFoldString(cte, "binary") {
		binary = true
	} else {
		return fmt.Errorf("unknown Content-Transfer-Encoding: %s", cte)
	}

	_ = binary

	if !ismultipart {
		if strings.HasPrefix(info.ContentType, "text/") {
			// try processing as main text
			// TODO make configurable
			const defaultTextBuf = 2048
			const maxTextBuf = (32 << 10) - 1
			b := bytes.NewBuffer(make([]byte, 0, defaultTextBuf))
			n, err := io.CopyN(b, r, maxTextBuf+1)
			if err != nil {
				return fmt.Errorf("error reading body: %v", err)
			}
			if n <= maxTextBuf {
				// it fit
				panic("TODO")
				return nil
			} else {
				// didn't fit, therefore we can't use this as main text
				r = io.MultiReader(b, r)
				// proceed with attachment processing
			}
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
