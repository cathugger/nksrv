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

	au "nksrv/lib/asciiutils"
	"nksrv/lib/emime"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/fstore"
	"nksrv/lib/ftypes"
	ht "nksrv/lib/hashtools"
	"nksrv/lib/mail"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
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
	_ io.Reader, rstr string, finished, preservemsgattachment bool, err error) {

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

		{
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

				// at this point message is readily edible

				// does it have nil chars?
				if strings.IndexByte(str, 0) < 0 {
					// normal processing - no need to have copy
					if !binary {
						// trim unneeded trailing newline without violating format
						str = au.TrimUnixNL(str)
					}
					return r, str, true, false, nil
				} else {
					// if contains nil characters, we need to modify it..
					rstr = tu.NormalizeTextMessage(strings.TrimRight(str, "\000"))
					preservemsgattachment = true
					// keep going, we need to preserve content
				}
			} else if cset == "" {
				// fallback
				cset = cfg.EmptyCharset
			}
		}

		// attempt to decode, if still haven't
		if !preservemsgattachment && cset != "" && !UorA {
			cod, e := ianaindex.MIME.Encoding(cset)
			if e == nil {
				dec := cod.NewDecoder()
				dstr, e := dec.String(str)
				// don't care about nil chars - normalization will rid of them
				if e == nil {
					// we don't care about binary mode
					// because this is just converted copy
					// so might aswell normalize and optimize it further
					rstr = tu.NormalizeTextMessage(dstr)
					preservemsgattachment = true
					// proceed with processing as attachment
				}
			}
		}

		// since we've read whole string, don't chain
		r = strings.NewReader(str)

	} else {
		// can't put in message
		// proceed with attachment processing
		r = io.MultiReader(strings.NewReader(str), r)
	}

	return r, rstr, false, preservemsgattachment, nil
}

func takeInFile(
	src *fstore.FStore, thm thumbnailer.ThumbExec, nothumb bool,
	ext, ctype string, r io.Reader, binary bool, ow io.Writer) (
	fn, hashname string, fsize int64,
	tres thumbnailer.ThumbResult, err error) {

	// new
	f, err := src.NewFile("tmp", "mail-", "")
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
	hashname, err = ht.MakeFileHash(f)
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
		tres, err = thm.ThumbProcess(f, ext, ctype, thm.ThumbConfig)
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
	ext = fu.SafeExt(oname)

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

type pmactx struct {
	*dmbctx

	nothumb  bool
	r        io.Reader
	binary   bool
	ct_t     string
	ct_par   map[string]string
	cdis_par map[string]string
	ow       io.Writer
}

func (ctx pmactx) processMessageAttachment() (
	fi FileInfo, fn string, err error) {

	// file extension, original name, corrected type
	var ext, oname string
	ext, oname, ctx.ct_t =
		attachmentInfo(ctx.ct_t, ctx.ct_par, ctx.cdis_par)

	// processing of file itself
	fn, hashname, fsize, tres, err := takeInFile(
		ctx.src, ctx.thmexec, ctx.nothumb,
		ext, ctx.ct_t, ctx.r, ctx.binary, ctx.ow)
	if err != nil {
		return
	}

	// ohwell, at this point we should probably have something
	// even if we don't, that's okay
	if ext != "" {
		hashname += "." + ext
	}
	// don't make up original name, it's ok not to have anything in it
	//if oname == "" {
	//	oname = hashname
	//}

	fi = FileInfo{
		ContentType: ctx.ct_t,
		Size:        fsize,
		ID:          hashname,
		Original:    oname,
	}

	fi.Type = tres.FI.Kind
	if tres.FI.DetectedType != "" {
		fi.ContentType = tres.FI.DetectedType
	}
	if tres.DBSuffix != "" {
		fi.ThumbField = ctx.thmexec.DBSuffix(tres.DBSuffix)
		fi.ThumbAttrib.Width = uint32(tres.Width)
		fi.ThumbAttrib.Height = uint32(tres.Height)

		ctx.thumbinfos = append(ctx.thumbinfos, ThumbInfo{
			FullTmpName: tres.CF.FullTmpName,
			RelDestName: ctx.thmexec.RelDestName(hashname, tres.CF.Suffix),
		})
		for _, ce := range tres.CE {
			ctx.thumbinfos = append(ctx.thumbinfos, ThumbInfo{
				FullTmpName: ce.FullTmpName,
				RelDestName: ctx.thmexec.RelDestName(hashname, ce.Suffix),
			})
		}
	}
	if len(tres.FI.Attrib) != 0 {
		fi.FileAttrib = tres.FI.Attrib
	}

	return
}

func ProcessContentType(ct string) (ct_t string, ct_par map[string]string) {
	if ct != "" {
		var e error
		ct_t, ct_par, e = mime.ParseMediaType(ct)
		// only replace with "invalid" if failed to parse completely
		if e != nil && ct_t == "" {
			ct_t = "invalid"
		}
	}
	return
}

// for shits possibly shared with deeper functions and shits what need to be changed
type dmbctx struct {
	cfg          *MailProcessorConfig
	src          *fstore.FStore
	thmexec      thumbnailer.ThumbExec
	thumbinfos   []ThumbInfo
	tmpfilenames []string
}

// for read-only things only used one level deep
type dmbin struct {
	ZH       mail.Headers
	zct_t    string
	zct_par  map[string]string
	eatinner bool
	zr       io.Reader
	oiw      io.Writer
}

// DevourMessageBody processes message body, filling in PostInfo structure,
// creating relevant files.
// It removes "Content-Transfer-Encoding" header from ZH,
// also sometimes modifies "Content-Type" header.
// info.ContentParams must be non-nil only if info.ContentType requires
// processing of params (text/*, multipart/*).
func (cfg *MailProcessorConfig) DevourMessageBody(
	src *fstore.FStore, thmexec thumbnailer.ThumbExec,
	ZH mail.Headers, zct_t string, zct_par map[string]string, eatinner bool,
	zr io.Reader, oiw io.Writer) (
	pi PostInfo, tmpfilenames []string, thumbinfos []ThumbInfo,
	IH mail.Headers, zerr error) {

	ctx := dmbctx{
		cfg:     cfg,
		src:     src,
		thmexec: thmexec,
	}
	zin := dmbin{
		ZH:       ZH,
		zct_t:    zct_t,
		zct_par:  zct_par,
		eatinner: eatinner,
		zr:       zr,
		oiw:      oiw,
	}
	pi, IH, zerr = ctx.devourMessageBody(zin)
	tmpfilenames = ctx.tmpfilenames
	thumbinfos = ctx.thumbinfos
	return
}

func (ctx *dmbctx) devourMessageBody(zin dmbin) (
	pi PostInfo, IH mail.Headers, zerr error) {

	defer func() {
		if zerr != nil {
			for _, fn := range ctx.tmpfilenames {
				if fn != "" {
					os.Remove(fn)
				}
			}
			ctx.tmpfilenames = nil

			for _, ti := range ctx.thumbinfos {
				if ti.FullTmpName != "" {
					os.Remove(ti.FullTmpName)
				}
			}
			ctx.thumbinfos = nil
		}
	}()

	// TODO parse multiple levels of multipart
	// TODO be picky about stuff from multipart/alternative, prefer txt, richtext

	// whether we already filled in .Message
	textprocessed := false

	guttleBody := func(
		r io.Reader, H mail.Headers, ct_t string, ct_par map[string]string,
		binary bool) (obj BodyObject, err error) {

		// attachment is original of msg field what must be preserved, msg was decoded
		preservemsgattachment := false

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
			r, str, finished, preservemsgattachment, err =
				ctx.cfg.processMessageText(r, binary, ct_t, ct_par)
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
			} else if !preservemsgattachment {
				// incase msg is filled use msg not attachment content
				pi.E.TextAttachment = uint32(len(pi.FI) + 1)
			}
		}

		// if this point is reached, we'll need to add this as attachment

		pmac := pmactx{
			dmbctx:   ctx,
			nothumb:  preservemsgattachment,
			r:        r,
			binary:   binary,
			ct_t:     ct_t,
			ct_par:   ct_par,
			cdis_par: cdis_par,
			ow:       nil,
		}
		fi, fn, err := pmac.processMessageAttachment()
		if err != nil {
			return
		}
		// this one is left here because signature code uses this
		ctx.tmpfilenames = append(ctx.tmpfilenames, fn)

		if preservemsgattachment {
			// if translated message was already stored in msg field
			fi.Type = ftypes.FTypeMsg
		}
		// save orig CT
		fi.Extras.ContentType = ct_t
		// add
		pi.FI = append(pi.FI, fi)

		obj.Data = PostObjectIndex(len(pi.FI))
		return
	}

	trackedGuttleBody := func(
		r io.Reader, H mail.Headers, ct_t string, ct_par map[string]string,
		binary bool) (obj BodyObject, hasNull, has8Bit bool, err error) {

		var rt *ReadTracker
		if !binary {
			rt = &ReadTracker{R: r}
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
		xct_t string, xct_par map[string]string, xcte string,
		XH mail.Headers, xr io.Reader) (
		rpinfo PartInfo, err error) {

		xismultipart := strings.HasPrefix(xct_t, "multipart/")

		var xbinary bool
		xr, _, xbinary, err =
			ctx.cfg.processMessagePrepareReader(xcte, xismultipart, xr)
		if err != nil {
			return
		}

		allowmpparam := func() bool {
			mpt := xct_t[len("multipart/"):]
			switch mpt {
			case "mixed", "alternative", "digest", "parallel", "form-data",
				"multilingual":
				// we know these have no more params
				return false
			}
			return true
		}

		if xismultipart && xct_par["boundary"] != "" &&
			XH.GetFirst("Content-Disposition") == "" {

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
					ctx.cfg.processMessagePrepareReader(
						pcte, pismultipart, pr)
				if err != nil {
					// XXX maybe skip only this part?
					return
				}

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

			// params of this go elsewhere
			XH["Content-Type"][0].V =
				au.TrimWSString(au.UntilString(XH["Content-Type"][0].V, ';'))
			// no need for this param after parsing
			// XXX maybe clone instead of modifying given?
			delete(xct_par, "boundary") // never include this there
			if len(xct_par) != 0 && allowmpparam() {
				rpinfo.MPParams = xct_par
			}

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
		if err != nil {
			err = fmt.Errorf("trackedGuttleBody: %v", err)
			return
		}

		return
	}

	eatMainAndHeaders := func(
		xct_t string, xct_par map[string]string, xcte string,
		XH mail.Headers, xr io.Reader) (
		rpinfo PartInfo, err error) {

		rpinfo, err = eatMain(xct_t, xct_par, xcte, XH, xr)
		if err != nil {
			return
		}

		// process face-like headers if any
		ffn, ffi, err := extractMessageFace(XH, ctx.src)
		if err != nil {
			err = fmt.Errorf("extractMessageFace: %v", err)
			return
		}
		if ffn != "" {
			ctx.tmpfilenames = append(ctx.tmpfilenames, ffn)
			pi.FI = append(pi.FI, ffi)
		}

		return
	}

	zcte := zin.ZH.GetFirst("Content-Transfer-Encoding")
	// we won't need this anymore
	delete(zin.ZH, "Content-Transfer-Encoding")

	if !zin.eatinner {

		// eat body
		pi.L, zerr = eatMainAndHeaders(
			zin.zct_t, zin.zct_par, zcte, zin.ZH, zin.zr)

	} else {
		// special handling for message/* bodies

		var ir io.Reader
		var ibinary bool
		ir, _, ibinary, zerr =
			ctx.cfg.processMessagePrepareReader(zcte, false, zin.zr)
		if zerr != nil {
			return
		}

		// additional worker which processes message without interpretation
		pir, piw := io.Pipe()
		ir = io.TeeReader(ir, piw)
		var wfi FileInfo
		var wfn string
		var werr error
		var wHasNull bool
		var wHas8Bit bool
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			var r io.Reader = pir
			var rt *ReadTracker
			if !ibinary {
				rt = &ReadTracker{R: r}
				r = rt
			}

			// this deals with body itself
			wpmac := pmactx{
				dmbctx:   ctx,
				nothumb:  true,
				r:        r,
				binary:   ibinary,
				ct_t:     zin.zct_t,
				ct_par:   zin.zct_par,
				cdis_par: nil,
				ow:       zin.oiw,
			}
			wfi, wfn, werr = wpmac.processMessageAttachment()

			if rt != nil {
				wHasNull = rt.HasNull
				wHas8Bit = rt.Has8Bit && !rt.HasNull
			}

			// keep on consuming to avoid deadlock incase worker is one who failed
			if werr != nil {
				_, _ = io.Copy(ioutil.Discard, pir)
			}

			wg.Done()
		}()
		cancelWorker := func(e error) {
			piw.CloseWithError(e)

			wg.Wait()

			if wfn != "" {
				os.Remove(wfn)
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
		_, zerr = eatMainAndHeaders(ict_t, ict_par, icte, IH, ir)
		if zerr != nil {
			zerr = fmt.Errorf("err eatin inner body: %v", zerr)
			cancelWorker(zerr)
			return
		}

		// ensure we read everything
		_, zerr = io.Copy(ioutil.Discard, ir)
		if zerr != nil {
			zerr = fmt.Errorf("err eatin remains of inner body: %v", zerr)
			cancelWorker(zerr)
			return
		}
		// signal worker that we're done
		piw.Close()
		// wait for worker
		wg.Wait()

		if werr != nil {
			zerr = fmt.Errorf("worker err eatin inner msg: %v", werr)
			return
		}

		wfi.Type = ftypes.FTypeMsg
		// add to tmp filenames
		ctx.tmpfilenames = append(ctx.tmpfilenames, wfn)
		// to fileinfos
		pi.FI = append(pi.FI, wfi)
		// set up proper body layout info
		pi.L.Body.Data = PostObjectIndex(len(pi.FI))
		pi.L.HasNull = wHasNull
		pi.L.Has8Bit = wHas8Bit
		// phew all done
		// (hopefuly I did everything right)
	}

	return
}

func DevourMessageBody(
	src *fstore.FStore, thm thumbnailer.ThumbExec,
	XH mail.Headers, xct_t string, xct_par map[string]string, eatinner bool,
	xr io.Reader, oiw io.Writer) (
	pi PostInfo, tmpfilenames []string, thmis []ThumbInfo,
	IH mail.Headers, err error) {

	return DefaultMailProcessorConfig.DevourMessageBody(
		src, thm, XH, xct_t, xct_par, eatinner, xr, oiw)
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
