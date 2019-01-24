package mailib

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	qp "mime/quotedprintable"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/encoding/ianaindex"

	au "centpd/lib/asciiutils"
	"centpd/lib/emime"
	"centpd/lib/fstore"
	"centpd/lib/ftypes"
	ht "centpd/lib/hashtools"
	"centpd/lib/mail"
	tu "centpd/lib/textutils"
	"centpd/lib/thumbnailer"
)

const DefaultHeaderSizeLimit = 2 << 20

type MailProcessorConfig struct {
	TryUTF8      bool   // whether we should try decoding unspecified charset as UTF8
	AllowBinary  bool   // whether we should allow "binary" Content-Transfer-Encoding
	EmptyCharset string // what encoding should we try if charset is unspecified
	MaxTextLen   uint   // maximum size of text message
}

const DefaultMaxTextLen = (64 << 10) - 1

var DefaultMailProcessorConfig = MailProcessorConfig{
	TryUTF8:      true,
	AllowBinary:  false,
	EmptyCharset: "ISO-8859-1",
	MaxTextLen:   DefaultMaxTextLen,
}

func (cfg *MailProcessorConfig) processMessagePrepareReader(
	cte string, ismultipart bool, r io.Reader) (
	_ io.Reader, translated, binary bool, err error) {

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
		translated = true
	} else if au.EqualFoldString(cte, "quoted-printable") {
		if ismultipart {
			err = errors.New("multipart x quoted-printable not allowed")
			return
		}
		r = qp.NewReader(r)
		binary = false
		translated = true
	} else if au.EqualFoldString(cte, "binary") && cfg.AllowBinary {
		binary = true
	} else {
		err = fmt.Errorf("unknown Content-Transfer-Encoding: %s", cte)
		return
	}
	return r, translated, binary, err
}

func (cfg *MailProcessorConfig) processMessageText(
	r io.Reader, binary bool, ct_t string, ct_par map[string]string) (
	_ io.Reader, rstr string, finished bool, msgattachment bool, err error) {

	// TODO maybe make configurable?
	const defaultTextBuf = 512

	b := &strings.Builder{}
	b.Grow(defaultTextBuf)

	if !binary {
		r = au.NewUnixTextReader(r)
	}
	n, err := io.CopyN(b, r, int64(cfg.MaxTextLen+1))
	if err != nil && err != io.EOF {
		err = fmt.Errorf("error reading body: %v", err)
		return
	}

	str := b.String()
	if n <= int64(cfg.MaxTextLen) {
		// it fit
		cset := ""
		if ct_t != "" && ct_par != nil {
			cset = ct_par["charset"]
		}

		UorA := au.EqualFoldString(cset, "UTF-8") ||
			au.EqualFoldString(cset, "US-ASCII")

		if strings.IndexByte(str, 0) < 0 {

			EorUorA := UorA || (cset == "" && cfg.TryUTF8)
			// expect UTF-8 in most cases
			if (EorUorA && utf8.ValidString(str)) ||
				(!EorUorA &&
					(cset == "" ||
						// ISO-8859-, Windows- and KOI8- variants are
						// US-ASCII compatible
						au.StartsWithFoldString(cset, "ISO-8859-") ||
						au.StartsWithFoldString(cset, "Windows-") ||
						au.StartsWithFoldString(cset, "KOI8-")) &&
					// so in case they don't use any extended characters,
					// they're effectively US-ASCII
					au.Is7BitString(str)) {

				// normal processing - no need to have copy
				if !binary {
					// trim unneeded trailing newline without violating format
					str = au.TrimUnixNL(str)
				}
				return r, str, true, false, nil

			} else if cset == "" {
				// fallback
				cset = cfg.EmptyCharset
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
	src *fstore.FStore, thm thumbnailer.ThumbExec, nothumb bool,
	ext, ctype string, r io.Reader, binary bool, ow io.Writer) (
	fn, hash, hashtype string, fsize int64,
	tres thumbnailer.ThumbResult, tfi thumbnailer.FileInfo, err error) {

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
			f.Close() // double close isn't much harm
			os.Remove(fn)
		}
	}()
	// copy
	if !binary {
		r = au.NewUnixTextReader(r)
	}
	fsize, err = io.Copy(f, r)
	if err != nil {
		return
	}
	// seek to 0
	_, err = f.Seek(0, 0)
	if err != nil {
		return
	}
	// TODO? io.MultiWriter?
	// hash it
	hash, hashtype, err = ht.MakeFileHash(f)
	if err != nil {
		return
	}
	if ow != nil {
		// seek to 0
		_, err = f.Seek(0, 0)
		if err != nil {
			return
		}
		// copy
		_, err = io.Copy(ow, f)
		if err != nil {
			return
		}
	}

	if !nothumb {
		// thumbnail (and also close)
		tres, tfi, err = thm.ThumbProcess(f, ext, ctype, thm.ThumbConfig)
	} else {
		// just close
		err = f.Close()
	}
	if err != nil {
		return
	}

	return
}

func attachmentInfo(
	ct_t string, ct_par, cdis_par map[string]string) (
	ext, oname, _ string) {

	// content-disposition filename param
	if cdis_par != nil && cdis_par["filename"] != "" {
		oname = cdis_par["filename"]
		// undo RFC 2047 MIME Word hackery, if any
		tr_oname, e := mail.DecodeMIMEWordHeader(oname)
		if e == nil {
			oname = tr_oname
		}
	}
	// content-type name param
	if oname == "" && ct_par != nil && ct_par["name"] != "" {
		oname = ct_par["name"]
		// undo RFC 2047 MIME Word hackery, if any
		tr_oname, e := mail.DecodeMIMEWordHeader(oname)
		if e == nil {
			oname = tr_oname
		}
	}
	// ensure oname is clean
	// we don't care about windows backslash because it wouldn't be harmful for us
	// since it's only used for display purposes
	if i := strings.LastIndexByte(oname, '/'); i >= 0 {
		oname = oname[i+1:]
	}

	// ext
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
	// correction
	if ext != "" {
		ext = emime.MIMEPreferedExtension(ext)
	}

	return ext, oname, ct_t
}

func processMessageAttachment(
	src *fstore.FStore, thm thumbnailer.ThumbExec, nothumb bool, r io.Reader,
	binary bool, ct_t string, ct_par, cdis_par map[string]string, ow io.Writer) (
	fi FileInfo, fn, thmfn string, err error) {

	// file extension, original name, corrected type
	ext, oname, ct_t := attachmentInfo(ct_t, ct_par, cdis_par)

	// processing of file itself
	fn, hash, hashtype, fsize, tres, tfi, err :=
		takeInFile(src, thm, nothumb, ext, ct_t, r, binary, ow)
	if err != nil {
		return
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

	if tres.FileName != "" {
		tfile := iname + "." + thm.Name + "." + tres.FileExt
		fi.Thumb = tfile
		fi.ThumbAttrib.Width = uint32(tres.Width)
		fi.ThumbAttrib.Height = uint32(tres.Height)
		fi.Type = tfi.Kind
		thmfn = tres.FileName
	}
	if len(tfi.Attrib) != 0 {
		fi.FileAttrib = tfi.Attrib
	}

	return
}

func ProcessContentType(ct string) (ct_t string, ct_par map[string]string) {
	if ct != "" {
		var e error
		ct_t, ct_par, e = mime.ParseMediaType(ct)
		if e != nil && ct_t == "" {
			ct_t = "invalid"
		}
	}
	return
}

// DevourMessageBody processes message body, filling in PostInfo structure,
// creating relevant files.
// It removes "Content-Transfer-Encoding" header from ZH,
// also sometimes modifies "Content-Type" header.
// info.ContentParams must be non-nil only if info.ContentType requires
// processing of params (text/*, multipart/*).
func (cfg *MailProcessorConfig) DevourMessageBody(
	src *fstore.FStore, thm thumbnailer.ThumbExec,
	ZH mail.Headers, zct_t string, zct_par map[string]string, zr io.Reader) (
	pi PostInfo, tmpfilenames []string, thumbfilenames []string, zerr error) {

	defer func() {
		if zerr != nil {
			for _, fn := range tmpfilenames {
				if fn != "" {
					os.Remove(fn)
				}
			}
			tmpfilenames = nil

			for _, fn := range thumbfilenames {
				if fn != "" {
					os.Remove(fn)
				}
			}
			thumbfilenames = nil
		}
	}()

	// TODO parse multiple levels of multipart
	// TODO be picky about stuff from multipart/alternative, prefer txt, richtext

	// whether we already filled in .Message
	textprocessed := false

	guttleBody := func(
		r io.Reader, H mail.Headers, ct_t string, ct_par map[string]string,
		binary bool) (obj BodyObject, err error) {

		// is used when message is properly decoded
		msgattachment := false

		cdis := H.GetFirst("Content-Disposition")
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
				cfg.processMessageText(r, binary, ct_t, ct_par)
			if err != nil {
				return
			}
			pi.MI.Message = str
			if finished {
				if str != "" {
					obj.Data = PostObjectIndex(0)
				} else {
					obj.Data = nil
				}
				return
			}
		}

		// if this point is reached, we'll need to add this as attachment

		fi, fn, thmfn, err := processMessageAttachment(
			src, thm, msgattachment, r, binary, ct_t, ct_par, cdis_par, nil)
		if err != nil {
			return
		}
		tmpfilenames = append(tmpfilenames, fn)
		thumbfilenames = append(thumbfilenames, thmfn)

		if msgattachment {
			// if translated message was already stored in msg field
			fi.Type = ftypes.FTypeMsg
		}
		pi.FI = append(pi.FI, fi)

		obj.Data = PostObjectIndex(len(pi.FI))
		return
	}

	trackedGuttleBody := func(
		r io.Reader, H mail.Headers, ct_t string, ct_par map[string]string,
		binary bool) (obj BodyObject, hasNull, has8Bit bool, err error) {

		var rt *readTracker
		if !binary {
			rt = &readTracker{R: r}
			r = rt
		}

		obj, err = guttleBody(r, H, ct_t, ct_par, binary)

		if rt != nil {
			hasNull = rt.HasNull
			has8Bit = rt.Has8Bit && !rt.HasNull
		}

		return
	}

	eatMain := func(
		xct_t string, xct_par map[string]string, xcte string, XH mail.Headers, xr io.Reader) (
		rpinfo PartInfo, err error) {

		xismultipart := strings.HasPrefix(xct_t, "multipart/")

		var xbinary bool
		xr, _, xbinary, err =
			cfg.processMessagePrepareReader(xcte, xismultipart, xr)
		if err != nil {
			return
		}

		if xismultipart && xct_par != nil && xct_par["boundary"] != "" &&
			len(XH["Content-Disposition"]) == 0 {

			has8bit := false

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

				pct := PH.GetFirst("Content-Type")
				// this will go elsewhere
				delete(PH, "Content-Type")
				pct_t, pct_par := ProcessContentType(pct)

				pcte := PH.GetFirst("Content-Transfer-Encoding")
				// we won't need this anymore
				delete(PH, "Content-Transfer-Encoding")

				pismultipart := strings.HasPrefix(pct_t, "multipart/")

				var pxr io.Reader
				var pbinary bool
				pxr, _, pbinary, err =
					cfg.processMessagePrepareReader(pcte, pismultipart, pr)

				var partI PartInfo
				partI.ContentType = pct
				partI.Binary = pbinary
				partI.Headers = PH

				partI.Body, partI.HasNull, partI.Has8Bit, err =
					trackedGuttleBody(pxr, PH, pct_t, pct_par, pbinary)
				if err != nil {
					err = fmt.Errorf("guttleBody: %v", err)
					break
				}

				if partI.Has8Bit {
					has8bit = true
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
			rpinfo.Body.Data = pis
			rpinfo.Binary = xbinary
			rpinfo.Has8Bit = has8bit

			return // done there
		}

		// if we reached this point we're not doing multipart

		// since this is toplvl dont set ContentType or other headers
		rpinfo.Binary = xbinary

		rpinfo.Body, rpinfo.HasNull, rpinfo.Has8Bit, err =
			trackedGuttleBody(xr, XH, xct_t, xct_par, xbinary)

		return
	}

	zcte := ZH.GetFirst("Content-Transfer-Encoding")
	// we won't need this anymore
	delete(ZH, "Content-Transfer-Encoding")

	if !strings.HasPrefix(zct_t, "message/") ||
		len(ZH["Content-Disposition"]) != 0 {

		// eat body
		pi.L, zerr = eatMain(zct_t, zct_par, zcte, ZH, zr)

	} else {
		// special handling for message/* bodies

		var ir io.Reader
		var IH mail.Headers
		var ibinary bool
		ir, _, ibinary, zerr = cfg.processMessagePrepareReader(zcte, false, zr)
		if zerr != nil {
			return
		}

		// additional worker which processes message without interpretation
		pir, piw := io.Pipe()
		ir = io.TeeReader(ir, piw)
		var wfi FileInfo
		var wfn, wthmfn string
		var werr error
		var wHasNull bool
		var wHas8Bit bool
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			var r io.Reader = pir
			var rt *readTracker
			if !ibinary {
				rt = &readTracker{R: r}
				r = rt
			}

			// this deals with body itself
			wfi, wfn, wthmfn, werr = processMessageAttachment(
				src, thm, true, r, ibinary, zct_t, zct_par, nil, nil)

			if rt != nil {
				wHasNull = rt.HasNull
				wHas8Bit = rt.Has8Bit && !rt.HasNull
			}

			// keep on consuming to avoid deadlock incase worker is one who failed
			if werr != nil {
				io.Copy(ioutil.Discard, pir)
			}

			wg.Done()
		}()
		cancelWorker := func(e error) {
			piw.CloseWithError(e)

			wg.Wait()

			if wfn != "" {
				os.Remove(wfn)
			}
			if wthmfn != "" {
				os.Remove(wthmfn)
			}
		}

		// interpret message
		// TODO configurable header size limit?
		var IMH mail.MessageHead
		IMH, zerr = mail.ReadHeaders(ir, 8<<10)
		if zerr != nil {
			zerr = fmt.Errorf("err readin inner message headers: %v", zerr)
			cancelWorker(zerr)
			return
		}
		defer IMH.Close()

		ir = IMH.B
		IH = IMH.H

		ict_t, ict_par := ProcessContentType(IH.GetFirst("Content-Type"))

		icte := au.TrimWSString(IH.GetFirst("Content-Transfer-Encoding"))
		delete(IH, "Content-Transfer-Encoding") // we won't need this anymore

		// eat body
		// yeh we discard its layout lol
		_, zerr = eatMain(ict_t, ict_par, icte, IH, ir)
		if zerr != nil {
			zerr = fmt.Errorf("err eatin inner body: %v", zerr)
			cancelWorker(zerr)
			return
		}

		// wait for worker
		wg.Wait()

		if werr != nil {
			zerr = fmt.Errorf("worker err eatin inner msg: %v", werr)
			return
		}

		wfi.Type = ftypes.FTypeMsg
		// insert into tmp filenames
		tmpfilenames = append(tmpfilenames, "")
		copy(tmpfilenames[1:], tmpfilenames)
		tmpfilenames[0] = wfn
		// into tmp thm filenames
		thumbfilenames = append(thumbfilenames, "")
		copy(thumbfilenames[1:], thumbfilenames)
		thumbfilenames[0] = wthmfn
		// into fileinfos
		pi.FI = append(pi.FI, FileInfo{})
		copy(pi.FI[1:], pi.FI)
		pi.FI[0] = wfi
		// set up proper body layout info
		pi.L.Body.Data = PostObjectIndex(0)
		pi.L.HasNull = wHasNull
		pi.L.Has8Bit = wHas8Bit
		// phew all done
		// (hopefuly I did everything right)
	}

	return
}

func DevourMessageBody(
	src *fstore.FStore, thm thumbnailer.ThumbExec,
	XH mail.Headers, xct_t string, xct_par map[string]string, xr io.Reader) (
	pi PostInfo, tmpfilenames []string, thumbfilenames []string, err error) {

	return DefaultMailProcessorConfig.DevourMessageBody(src, thm, XH, xct_t, xct_par, xr)
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
