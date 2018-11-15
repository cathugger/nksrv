package mailib

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

	"golang.org/x/text/encoding/ianaindex"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/emime"
	"nekochan/lib/fstore"
	ht "nekochan/lib/hashtools"
	"nekochan/lib/mail"
	tu "nekochan/lib/textutils"
)

func processMessagePrepareReader(
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

func processMessageText(
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
		if ct != "" && cpar != nil {
			cset = cpar["charset"]
		}

		UorA := au.EqualFoldString(cset, "UTF-8") ||
			au.EqualFoldString(cset, "US-ASCII")

		if strings.IndexByte(str, 0) < 0 {

			EorUorA := cset == "" || UorA

			// expect UTF-8 in most cases
			if (EorUorA && utf8.ValidString(str)) ||
				(!EorUorA && // all ISO-8859-* are ASCII compatible
					au.StartsWithFoldString(cset, "ISO-8859-") &&
					au.Is7BitString(str)) {

				// normal processing - no need to have copy
				if !binary {
					str = au.TrimUnixNL(str)
				}
				return r, str, true, false, nil

			} else if cset == "" {
				// fallback to ISO-8859-1
				cset = "ISO-8859-1"
			}
		}

		// attempt to decode
		if cset != "" && !UorA {
			cod, e := ianaindex.MIME.Encoding(cset)
			if e == nil {
				dec := cod.NewDecoder()
				dstr, e := dec.String(str)
				// should not result in null characters
				if e == nil && strings.IndexByte(dstr, 0) < 0 {
					// we don't care about binary mode
					// because this is just converted copy
					// so might aswell normalize and optimize it further
					rstr = tu.NormalizeTextMessage(dstr)
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

func processMessageAttachment(
	src *fstore.FStore, H mail.Headers, r io.Reader, binary bool, ct string) (
	fi FileInfo, fn string, err error) {

	// new
	f, err := src.TempFile("mail-", "")
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
	h, err := ht.MakeFileHash(f)
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
		cdis = H["Content-Disposition"][0].V
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
	// if still no extension, try treating "text/*" as "text/plain"
	if ext == "" && strings.HasPrefix(ct, "text/") && ct != "text/plain" {
		mexts, e := emime.MIMEExtensionsByType("text/plain")
		if e == nil && len(mexts) != 0 {
			ext = mexts[0] // expect first to be good enough
		}
	}
	// special fallbacks, better than nothing
	if ext == "" {
		if strings.HasPrefix(ct, "text/") ||
			strings.HasPrefix(ct, "multipart/") {

			ext = "txt"
		} else if strings.HasPrefix(ct, "message/") {
			ext = "eml"
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
	// don't make up original name, it's ok not to have anything in it
	//if oname == "" {
	//	oname = iname
	//}

	fi = FileInfo{
		ContentType: ct,
		Size:        n,
		ID:          iname,
		Original:    oname,
	}
	return
}

// DevourMessageBody processes message body, filling in PostInfo structure,
// creating relevant files.
// It removes "Content-Transfer-Encoding" header from XH,
// also sometimes modifies "Content-Type" header.
// info.ContentParams must be non-nil only if info.ContentType requires
// processing of params (text/*, multipart/*).
func DevourMessageBody(
	src *fstore.FStore, XH mail.Headers, info ParsedMessageInfo, xr io.Reader) (
	pi PostInfo, tmpfilenames []string, err error) {

	defer func() {
		if err != nil {
			for _, fn := range tmpfilenames {
				os.Remove(fn)
			}
			tmpfilenames = []string(nil)
		}
	}()

	// TODO parse multiple levels of multipart
	// TODO be picky about stuff from multipart/alternative, prefer txt, richtext

	textprocessed := false

	guttleBody := func(
		r io.Reader, H mail.Headers, ct string, cpar map[string]string,
		binary bool) (obj BodyObject, err error) {

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
				processMessageText(r, binary, ct, cpar)
			if err != nil {
				return
			}
			pi.MI.Message = str
			if finished {
				obj.Data = PostObjectIndex(0)
				return
			}
		}

		// if this point is reached, we'll need to add this as attachment

		fi, fn, err := processMessageAttachment(src, H, r, binary, ct)
		tmpfilenames = append(tmpfilenames, fn)
		if err != nil {
			return
		}

		if msgattachment {
			// if translated message was already stored in msg field
			fi.Type = FTypeMsg
		}
		pi.FI = append(pi.FI, fi)

		obj.Data = PostObjectIndex(len(pi.FI))
		return
	}

	xismultipart := strings.HasPrefix(info.ContentType, "multipart/")

	var xcte string
	if len(XH["Content-Transfer-Encoding"]) != 0 {
		xcte = XH["Content-Transfer-Encoding"][0].V
	}
	// we won't need this anymore
	delete(XH, "Content-Transfer-Encoding")

	var xbinary bool
	xr, xbinary, err =
		processMessagePrepareReader(xcte, xismultipart, xr)
	if err != nil {
		return
	}

	if xismultipart && info.ContentParams["boundary"] != "" &&
		len(XH["Content-Disposition"]) == 0 {

		pr := mail.NewPartReader(xr, info.ContentParams["boundary"])
		var pis []PartInfo
		for {
			err = pr.NextPart()
			if err != nil {
				break
			}

			var PH mail.Headers
			PH, err = pr.ReadHeaders(8 << 10)
			if err != nil {
				err = fmt.Errorf("pr.ReadHeaders: %v", err)
				break
			}

			var pct string
			if len(PH["Content-Type"]) != 0 {
				pct = PH["Content-Type"][0].V
			}
			delete(PH, "Content-Type")

			var pxct string
			var pxctparam map[string]string
			if pct != "" {
				var e error
				pxct, pxctparam, e = mime.ParseMediaType(pct)
				if e != nil {
					pxct = "invalid"
					pxctparam = map[string]string(nil)
				}
			}

			var pcte string
			if len(PH["Content-Transfer-Encoding"]) != 0 {
				pcte = PH["Content-Transfer-Encoding"][0].V
			}
			delete(PH, "Content-Transfer-Encoding")

			pismultipart := strings.HasPrefix(pct, "multipart/")

			var pxr io.Reader
			var pbinary bool
			pxr, pbinary, err =
				processMessagePrepareReader(pcte, pismultipart, pr)

			var partI PartInfo
			partI.ContentType = pct
			partI.Binary = pbinary
			partI.Headers = PH
			partI.Body, err =
				guttleBody(pxr, PH, pxct, pxctparam, pbinary)
			if err != nil {
				err = fmt.Errorf("guttleBody: %v", err)
				break
			}
			pis = append(pis, partI)
		}
		pr.Close()
		if err != io.EOF {
			err = fmt.Errorf("failed to parse multipart: %v", err)
			return
		}

		// no more parts
		err = nil
		// we're not going to save parameters of this
		XH["Content-Type"][0].V =
			au.TrimWSString(au.UntilString(XH["Content-Type"][0].V, ';'))
		// fill in
		pi.H = XH
		pi.L.Binary = xbinary
		pi.L.Body.Data = pis
		return // done there
	}

	// if we reached this point we're not doing multipart

	pi.H = XH
	// since this is toplvl dont set ContentType or other headers
	pi.L.Binary = xbinary
	pi.L.Body, err =
		guttleBody(xr, XH, info.ContentType, info.ContentParams, xbinary)

	return
}
