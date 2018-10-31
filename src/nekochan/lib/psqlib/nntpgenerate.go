package psqlib

import (
	crand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/mail"
)

func randomBoundary() string {
	var b [36]byte
	_, err := io.ReadFull(crand.Reader, b[:])
	if err != nil {
		panic(err)
	}
	return sBase64Enc.EncodeToString(b[:])
}

var errNoContentType = errors.New("no Content-Type")

func modifyMultipartType(ct string) (_ string, rb string, _ error) {
	rb = randomBoundary()
	if ct == "" {
		return "", "", errNoContentType
	}
	return ct + "; boundary=" + rb, rb, nil
}

func (sp *PSQLIB) nntpGenerate(
	xw io.Writer, num uint64, msgid CoreMsgIDStr) (err error) {

	// TODO
	// need to generate article off database shit
	// im too fucking lazy to actually do it atm
	// so placeholder to test shit will work for now

	// fetch info about post. some of info we don't care about
	q := `SELECT jp.title,jp.message,jp.headers,jp.layout,jf.fname
FROM ib0.posts AS jp
JOIN ib0.files AS jf
USING (bid,pid)
WHERE jp.msgid = $1
ORDER BY jf.fid`
	rows, err := sp.db.DB.Query(q, string(msgid))
	if err != nil {
		return sp.sqlError("posts x files query", err)
	}

	pi := postInfo{}

	for rows.Next() {
		var fi fileInfo

		// XXX is it okay to overwrite stuff there?
		err = rows.Scan(&pi.MI.Title, &pi.MI.Message, &pi.H, &pi.L, &fi.ID)
		if err != nil {
			rows.Close()
			return sp.sqlError("posts x files query rows scan", err)
		}

		if fi.ID != "" {
			pi.FI = append(pi.FI, fi)
		}
	}
	if err = rows.Err(); err != nil {
		return sp.sqlError("posts x files query rows iteration", err)
	}

	// ensure Message-ID
	if len(pi.H["Message-ID"]) == 0 {
		pi.H["Message-ID"] = []string{fmt.Sprintf("<%s>", msgid)}
	}

	// ensure Subject
	if len(pi.H["Subject"]) == 0 && pi.MI.Title != "" {
		pi.H["Subject"] = []string{pi.MI.Title}
	}

	generateBody := func(
		w io.Writer, binary bool, poi postObjectIndex) (err error) {

		var r io.Reader
		if poi == 0 {
			r = strings.NewReader(pi.MI.Message)
		} else {
			if uint(poi) > uint(len(pi.FI)) {
				panic("bad poi")
			}

			var f *os.File
			f, err = os.Open(sp.src.Main() + pi.FI[poi-1].ID)
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
		w io.Writer, binary bool, bo bodyObject) error {

		if poi, ok := bo.Data.(postObjectIndex); ok {
			return generateBody(w, binary, poi)
		}
		if bo.Data == nil {
			return nil
		}
		panic("bad bo.Data type")
	}

	var generateMultipart func(
		w io.Writer, boundary string, pis []partInfo) (err error)

	generateMultipart = func(
		w io.Writer, boundary string, pis []partInfo) (err error) {

		pw := mail.NewPartWriter(w, boundary, "")
		for i := range pis {
			ppis, ismp := pis[i].Body.Data.([]partInfo)
			var pb string
			if !ismp {
				if pis[i].ContentType != "" {
					pis[i].Headers["Content-Type"] = []string{pis[i].ContentType}
				}
				// XXX should we announce 8bit text?
				if pis[i].Binary {
					pis[i].Headers["Content-Transfer-Encoding"] =
						[]string{"base64"}
				}
			} else {
				pis[i].Headers["Content-Type"] = []string{""}
				pis[i].Headers["Content-Type"][0], pb, err =
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
				return generateSomething(w, pis[i].Binary, pis[i].Body)
			} else {
				return generateMultipart(w, pb, ppis)
			}
		}
		return pw.FinishParts("")
	}

	pis, ismp := pi.L.Body.Data.([]partInfo)
	var bnd string
	if !ismp {
		// XXX should we announce 8bit text?
		if pi.L.Binary {
			pi.H["Content-Transfer-Encoding"] = []string{"base64"}
		}
	} else {
		if len(pi.H["Content-Type"]) == 0 {
			return errNoContentType
		}
		pi.H["Content-Type"] = pi.H["Content-Type"][:1]
		pi.H["Content-Type"][0], bnd, err =
			modifyMultipartType(pi.H["Content-Type"][0])
		if err != nil {
			return
		}
	}
	mail.WriteHeaders(xw, pi.H, true)
	fmt.Fprintf(xw, "\n")

	if !ismp {
		return generateSomething(xw, pi.L.Binary, pi.L.Body)
	} else {
		return generateMultipart(xw, bnd, pis)
	}
}
