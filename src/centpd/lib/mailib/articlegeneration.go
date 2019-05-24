package mailib

import (
	crand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"

	au "centpd/lib/asciiutils"
	ht "centpd/lib/hashtools"
	"centpd/lib/mail"
)

var errNoContentType = errors.New("no Content-Type")

func randomBoundary() string {
	var b [33]byte
	_, err := io.ReadFull(crand.Reader, b[:])
	if err != nil {
		panic(err)
	}
	return ht.SBase64Enc.EncodeToString(b[:])
}

func modifyMultipartType(
	ct string, params map[string]string) (res string, rb string, _ error) {

	if ct == "" {
		return "", "", errNoContentType
	}

	rb = randomBoundary()

	if len(params) == 0 {
		// fast path
		return ct + "; boundary=" + rb, rb, nil
	} else {
		par := make(map[string]string)
		for k, v := range params {
			par[k] = v
		}
		par["boundary"] = rb
		res = mail.FormatMediaTypeX(ct, par)
		if res == "" {
			return "", "", errors.New("mail.FormatMediaTypeX failed")
		}
		return res, rb, nil
	}
}

func setCTE(
	H mail.Headers, pi PartInfo, w io.Writer) (
	_ mail.Headers, cw io.Writer, cc io.Closer) {

	cw = w // default

	if pi.Binary {
		if H == nil {
			H = make(mail.Headers)
		}
		H["Content-Transfer-Encoding"] = mail.OneHeaderVal("base64")

		ww := base64.NewEncoder(
			base64.StdEncoding, &au.SplitWriter{W: w, N: 76})
		cw = ww
		cc = ww
	} else if pi.HasNull {
		if H == nil {
			H = make(mail.Headers)
		}
		H["Content-Transfer-Encoding"] = mail.OneHeaderVal("quoted-printable")

		ww := quotedprintable.NewWriter(w)
		cw = ww
		cc = ww
	} else if pi.Has8Bit {
		if H == nil {
			H = make(mail.Headers)
		}
		H["Content-Transfer-Encoding"] = mail.OneHeaderVal("8bit")
	}

	return H, cw, cc
}

type InputFileList interface {
	OpenFileAt(i int) (io.ReadCloser, error)
}

func GenerateMessage(
	xw io.Writer, pi PostInfo, ifl InputFileList) (err error) {

	var xcw io.Writer
	var xcc io.Closer

	pis, ismp := pi.L.Body.Data.([]PartInfo)
	var mpboundary string
	if !ismp {
		pi.H, xcw, xcc = setCTE(pi.H, pi.L, xw)
	} else {
		if pi.H == nil || pi.H.GetFirst("Content-Type") == "" {
			return errNoContentType
		}

		if pi.L.Has8Bit && !pi.L.HasNull {
			pi.H["Content-Transfer-Encoding"] = mail.OneHeaderVal("8bit")
		}

		pi.H["Content-Type"] = pi.H["Content-Type"][:1]
		pi.H["Content-Type"][0].V, mpboundary, err =
			modifyMultipartType(pi.H["Content-Type"][0].V, pi.L.MPParams)
		if err != nil {
			return
		}
	}
	if pi.H != nil {
		mail.WriteHeaders(xw, pi.H, true)
	}
	fmt.Fprintf(xw, "\n")

	generateBody := func(
		w io.Writer, binary bool, poi PostObjectIndex) (err error) {

		var r io.Reader
		var lr *io.LimitedReader
		if poi == 0 {
			r = strings.NewReader(pi.MI.Message)
		} else {
			if uint(poi) > uint(len(pi.FI)) {
				panic("bad poi")
			}

			var f io.ReadCloser
			f, err = ifl.OpenFileAt(int(poi - 1))
			if err != nil {
				err = fmt.Errorf("failed to open file %d %q: %v",
					poi, pi.FI[poi-1].ID, err)
				return
			}
			defer f.Close()

			lr = &io.LimitedReader{R: f, N: pi.FI[poi-1].Size + 1}
			r = lr
		}

		if !binary {
			r = au.NewUnixTextReader(r)
		}

		_, err = io.Copy(w, r)
		if err != nil {
			err = fmt.Errorf("error copying: %v", err)
			return
		}
		if lr != nil && lr.N != 1 {
			err = errors.New("wrong amount copied")
			return
		}

		return
	}

	generateSomething := func(
		w io.Writer, c io.Closer, binary bool, bo BodyObject) (err error) {

		if poi, ok := bo.Data.(PostObjectIndex); ok {
			err = generateBody(w, binary, poi)
			if err != nil {
				return fmt.Errorf("generateBody err: %v", err)
			}
		} else if bo.Data == nil {
			// write nothing
		} else {
			panic("bad bo.Data type")
		}
		if c != nil {
			err = c.Close()
			if err != nil {
				return fmt.Errorf("error closing content writer: %v", err)
			}
		}
		return
	}

	var generateMultipart func(
		w io.Writer, boundary string, pis []PartInfo) (err error)

	generateMultipart = func(
		w io.Writer, boundary string, pis []PartInfo) (err error) {

		pw := mail.NewPartWriter(w, boundary, "")
		for i := range pis {
			var pcw io.Writer
			var pcc io.Closer

			ppis, ismp := pis[i].Body.Data.([]PartInfo)
			var pb string
			if !ismp {
				if pis[i].ContentType != "" {
					if pis[i].Headers == nil {
						pis[i].Headers = make(mail.Headers)
					}
					pis[i].Headers["Content-Type"] =
						mail.OneHeaderVal(pis[i].ContentType)
				}
				pis[i].Headers, pcw, pcc = setCTE(pis[i].Headers, pis[i], w)
			} else {
				if pis[i].ContentType == "" {
					return errNoContentType
				}

				if pis[i].Headers == nil {
					pis[i].Headers = make(mail.Headers)
				}

				if pis[i].Has8Bit && !pis[i].HasNull {
					pis[i].Headers["Content-Transfer-Encoding"] =
						mail.OneHeaderVal("8bit")
				}

				var ctv string
				ctv, pb, err =
					modifyMultipartType(pis[i].ContentType, pis[i].MPParams)
				if err != nil {
					return
				}
				pis[i].Headers["Content-Type"] = mail.OneHeaderVal(ctv)
			}

			err = pw.StartNextPart(pis[i].Headers)
			if err != nil {
				return
			}

			if !ismp {
				err = generateSomething(pcw, pcc, pis[i].Binary, pis[i].Body)
			} else {
				err = generateMultipart(w, pb, ppis)
			}
			if err != nil {
				return
			}
		}
		return pw.FinishParts("")
	}

	if !ismp {
		return generateSomething(xcw, xcc, pi.L.Binary, pi.L.Body)
	} else {
		return generateMultipart(xw, mpboundary, pis)
	}
}
