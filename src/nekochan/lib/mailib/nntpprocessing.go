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

const DefaultHeaderSizeLimit = 2 << 20

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
	r io.Reader, binary bool, ct_t string, ct_par map[string]string) (
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
		if ct_t != "" && ct_par != nil {
			cset = ct_par["charset"]
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

func takeInFile(
	src *fstore.FStore, r io.Reader, binary bool) (
	fn, hash, hashtype string, fsize int64, err error) {

	// new
	f, err := src.TempFile("mail-", "")
	if err != nil {
		return
	}
	// get name
	fn = f.Name()
	// cleanup on err
	defer func() {
		if err != nil {
			os.Remove(fn)
		}
	}()
	// copy
	if !binary {
		r = au.NewUnixTextReader(r)
	}
	fsize, err = io.Copy(f, r)
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
	hash, hashtype, err = ht.MakeFileHash(f)
	if err != nil {
		f.Close()
		return
	}

	// XXX do something more before closing file?

	// close
	err = f.Close()
	if err != nil {
		return
	}

	return
}

func processMessageAttachment(
	src *fstore.FStore, H mail.Headers, r io.Reader,
	binary bool, ct_t string, ct_par, cdis_par map[string]string) (
	fi FileInfo, fn string, err error) {

	fn, hash, hashtype, fsize, err := takeInFile(src, r, binary)
	if err != nil {
		return
	}

	// determine with what filename should we store it
	oname := ""
	if cdis_par != nil && cdis_par["filename"] != "" {
		oname = cdis_par["filename"]
		// undo RFC 2047 MIME Word hackery, if any
		tr_oname, e := mail.DecodeMIMEWordHeader(oname)
		if e == nil {
			oname = tr_oname
		}
		if !utf8.ValidString(oname) {
			oname = ""
		}
	}
	if oname == "" && ct_par != nil && ct_par["name"] != "" {
		oname = ct_par["name"]
		// undo RFC 2047 MIME Word hackery, if any
		tr_oname, e := mail.DecodeMIMEWordHeader(oname)
		if e == nil {
			oname = tr_oname
		}
		if !utf8.ValidString(oname) {
			oname = ""
		}
	}
	// ensure oname is clean
	// we don't care about windows backslash because it wouldn't be harmful for us
	// since it's only used for display purposes
	if i := strings.LastIndexByte(oname, '/'); i >= 0 {
		oname = oname[i+1:]
	}
	ext := ""
	if oname != "" {
		i := strings.LastIndexByte(oname, '.')
		// do some additional checks to ensure that extension at least makes sense
		// since we will be storing that in filesystem
		if i >= 0 && i+1 < len(oname) &&
			strings.IndexAny(oname[i+1:], "\\:*\"?<>|") < 0 &&
			!au.ContainsControlString(oname[i+1:]) {

			ext = oname[i+1:]
		}
	}
	if ct_t == "" {
		// default
		ct_t = "text/plain"
	}
	if ext == "" || !emime.MIMEIsCanonical(ext, ct_t) {
		// attempt finding better extension
		mexts, e := emime.MIMEExtensionsByType(ct_t)
		if e == nil && len(mexts) != 0 {
			ext = mexts[0] // expect first to be good enough
		}
	}
	// if still no extension, try treating "text/*" as "text/plain"
	if ext == "" && strings.HasPrefix(ct_t, "text/") && ct_t != "text/plain" {
		mexts, e := emime.MIMEExtensionsByType("text/plain")
		if e == nil && len(mexts) != 0 {
			ext = mexts[0] // expect first to be good enough
		}
	}
	// special fallbacks, better than nothing
	if ext == "" {
		if strings.HasPrefix(ct_t, "text/") ||
			strings.HasPrefix(ct_t, "multipart/") {

			ext = "txt"
		} else if strings.HasPrefix(ct_t, "message/") {
			ext = "eml"
		}
	}
	// ohwell, at this point we should probably have something
	// even if we don't, that's okay
	var iname string
	if ext != "" {
		iname = hash + "-" + hashtype + "." + ext
	} else {
		iname = hash + "-" + hashtype
	}
	// don't make up original name, it's ok not to have anything in it
	//if oname == "" {
	//	oname = iname
	//}

	fi = FileInfo{
		ContentType: ct_t,
		Size:        fsize,
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
		r io.Reader, H mail.Headers, ct_t string, ct_par map[string]string,
		binary bool) (obj BodyObject, err error) {

		// is used when message is properly decoded
		msgattachment := false

		cdis := ""
		if len(H["Content-Disposition"]) != 0 {
			cdis = H["Content-Disposition"][0].V
		}
		var cdis_t string
		var cdis_par map[string]string
		if cdis != "" {
			var e error
			cdis_t, cdis_par, e = mime.ParseMediaType(cdis)
			if e != nil {
				cdis_t = "invalid"
			}
		}

		if !textprocessed &&
			(ct_t == "" ||
				(strings.HasPrefix(ct_t, "text/") &&
					ct_par != nil && ct_par["name"] == "")) &&
			(cdis_t == "" ||
				(cdis_t == "inline" &&
					cdis_par != nil && cdis_par["filename"] == "")) {

			// try processing as main text
			// even if we fail, don't try doing it with other part
			textprocessed = true

			var str string
			var finished bool
			r, str, finished, msgattachment, err =
				processMessageText(r, binary, ct_t, ct_par)
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

		fi, fn, err :=
			processMessageAttachment(src, H, r, binary, ct_t, ct_par, cdis_par)
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

	var xct_t string
	var xct_par map[string]string
	if len(XH["Content-Type"]) != 0 {
		var e error
		ct := XH["Content-Type"][0].V
		xct_t, xct_par, e = mime.ParseMediaType(ct)
		if e != nil && xct_t == "" {
			xct_t = "invalid"
		}
	}

	xismultipart := strings.HasPrefix(xct_t, "multipart/")

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

	if xismultipart && xct_par != nil && xct_par["boundary"] != "" &&
		len(XH["Content-Disposition"]) == 0 {

		pr := mail.NewPartReader(xr, xct_par["boundary"])
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

			var pct_t string
			var pct_par map[string]string
			if pct != "" {
				var e error
				pct_t, pct_par, e = mime.ParseMediaType(pct)
				if e != nil && pct_t == "" {
					pct_t = "invalid"
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
				guttleBody(pxr, PH, pct_t, pct_par, pbinary)
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
		guttleBody(xr, XH, xct_t, xct_par, xbinary)

	return
}

func CleanContentTypeAndTransferEncoding(H mail.Headers) {
	// ignore other headers than first, trim whitespace
	if len(H["Content-Type"]) != 0 {
		ct := au.TrimWSString(H["Content-Type"][0].V)
		if ct != "" {
			H["Content-Type"] = H["Content-Type"][:1]
			H["Content-Type"][0].V = ct
		} else {
			delete(H, "Content-Type")
		}
	}
	if len(H["Content-Transfer-Encoding"]) != 0 {
		cte := au.TrimWSString(H["Content-Transfer-Encoding"][0].V)
		if cte != "" {
			H["Content-Transfer-Encoding"] = H["Content-Transfer-Encoding"][:1]
			H["Content-Transfer-Encoding"][0].V = cte
		} else {
			delete(H, "Content-Transfer-Encoding")
		}
	}
}
