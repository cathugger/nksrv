package mailib

import (
	crand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/fstore"
	ht "nekochan/lib/hashtools"
	"nekochan/lib/mail"
)

var errNoContentType = errors.New("no Content-Type")

func randomBoundary() string {
	var b [36]byte
	_, err := io.ReadFull(crand.Reader, b[:])
	if err != nil {
		panic(err)
	}
	return ht.SBase64Enc.EncodeToString(b[:])
}

func modifyMultipartType(ct string) (_ string, rb string, _ error) {
	rb = randomBoundary()
	if ct == "" {
		return "", "", errNoContentType
	}
	return ct + "; boundary=\"" + rb + "\"", rb, nil
}

func GenerateMessage(
	src *fstore.FStore, xw io.Writer, pi PostInfo) (err error) {

	pis, ismp := pi.L.Body.Data.([]PartInfo)
	var mpboundary string
	if !ismp {
		// XXX should we announce 8bit text?
		if pi.L.Binary {
			if pi.H == nil {
				pi.H = make(mail.Headers)
			}
			pi.H["Content-Transfer-Encoding"] = mail.OneHeaderVal("base64")
		}
	} else {
		if pi.H == nil || len(pi.H["Content-Type"]) == 0 {
			return errNoContentType
		}
		pi.H["Content-Type"] = pi.H["Content-Type"][:1]
		pi.H["Content-Type"][0].V, mpboundary, err =
			modifyMultipartType(pi.H["Content-Type"][0].V)
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
		if poi == 0 {
			r = strings.NewReader(pi.MI.Message)
		} else {
			if uint(poi) > uint(len(pi.FI)) {
				panic("bad poi")
			}

			var f *os.File
			f, err = os.Open(src.Main() + pi.FI[poi-1].ID)
			if err != nil {
				err = fmt.Errorf("failed to open file %d %q: %v",
					poi, pi.FI[poi-1].ID, err)
				return
			}
			defer f.Close()

			r = f
		}
		if !binary {
			r = au.NewUnixTextReader(r)
		} else {
			w = base64.NewEncoder(
				base64.StdEncoding, &au.SplitWriter{W: w, N: 116})
		}

		_, err = io.Copy(w, r)
		if err != nil {
			err = fmt.Errorf("error copying: %v", err)
		}
		return
	}

	generateSomething := func(
		w io.Writer, binary bool, bo BodyObject) error {

		if poi, ok := bo.Data.(PostObjectIndex); ok {
			return generateBody(w, binary, poi)
		}
		if bo.Data == nil {
			return nil
		}
		panic("bad bo.Data type")
	}

	var generateMultipart func(
		w io.Writer, boundary string, pis []PartInfo) (err error)

	generateMultipart = func(
		w io.Writer, boundary string, pis []PartInfo) (err error) {

		pw := mail.NewPartWriter(w, boundary, "")
		for i := range pis {
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
				// XXX should we announce 8bit text?
				if pis[i].Binary {
					if pis[i].Headers == nil {
						pis[i].Headers = make(mail.Headers)
					}
					pis[i].Headers["Content-Transfer-Encoding"] =
						mail.OneHeaderVal("base64")
				}
			} else {
				if pis[i].Headers == nil {
					pis[i].Headers = make(mail.Headers)
				}
				pis[i].Headers["Content-Type"] = []mail.HeaderVal{{}}
				pis[i].Headers["Content-Type"][0].V, pb, err =
					modifyMultipartType(pis[i].ContentType)
				if err != nil {
					return
				}
			}

			err = pw.StartNextPart(pis[i].Headers)
			if err != nil {
				return
			}

			if !ismp {
				err = generateSomething(w, pis[i].Binary, pis[i].Body)
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
		return generateSomething(xw, pi.L.Binary, pi.L.Body)
	} else {
		return generateMultipart(xw, mpboundary, pis)
	}
}
