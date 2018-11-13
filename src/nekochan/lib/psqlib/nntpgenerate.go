package psqlib

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	au "nekochan/lib/asciiutils"
	ht "nekochan/lib/hashtools"
	. "nekochan/lib/logx"
	"nekochan/lib/mail"
	"nekochan/lib/mailib"

	xtypes "github.com/jmoiron/sqlx/types"
)

func randomBoundary() string {
	var b [36]byte
	_, err := io.ReadFull(crand.Reader, b[:])
	if err != nil {
		panic(err)
	}
	return ht.SBase64Enc.EncodeToString(b[:])
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

	// fetch info about post. some of info we don't care about
	q := `SELECT jp.title,jp.message,jp.headers,jp.layout,jf.fname
FROM ib0.posts AS jp
LEFT JOIN ib0.files AS jf
USING (bid,pid)
WHERE jp.msgid = $1
ORDER BY jf.fid`
	rows, err := sp.db.DB.Query(q, string(msgid))
	if err != nil {
		return sp.sqlError("posts x files query", err)
	}

	pi := mailib.PostInfo{}

	havesomething := false

	for rows.Next() {
		havesomething = true

		var jH, jL xtypes.JSONText
		var fid sql.NullString

		// XXX is it okay to overwrite stuff there?
		err = rows.Scan(&pi.MI.Title, &pi.MI.Message, &jH, &jL, &fid)
		if err != nil {
			rows.Close()
			return sp.sqlError("posts x files query rows scan", err)
		}

		sp.log.LogPrintf(DEBUG,
			"nntpGenerate: PxF: title(%q) msg(%q) H(%q) L(%q) id(%v)",
			pi.MI.Title, pi.MI.Message, jH, jL, fid)

		err = jH.Unmarshal(&pi.H)
		if err != nil {
			rows.Close()
			return sp.sqlError("jH unmarshal", err)
		}

		err = jL.Unmarshal(&pi.L)
		if err != nil {
			rows.Close()
			return sp.sqlError("jL unmarshal", err)
		}

		sp.log.LogPrintf(DEBUG,
			"nntpGenerate: unmarshaled H(%#v) L(%#v)",
			pi.H, &pi.L)

		if fid.Valid && fid.String != "" {
			pi.FI = append(pi.FI, mailib.FileInfo{ID: fid.String})
		}
	}
	if err = rows.Err(); err != nil {
		return sp.sqlError("posts x files query rows iteration", err)
	}

	if !havesomething {
		return errNotExist
	}

	// ensure Message-ID
	if len(pi.H["Message-ID"]) == 0 {
		pi.H["Message-ID"] = mail.OneHeaderVal(fmt.Sprintf("<%s>", msgid))
	}

	// ensure Subject
	if len(pi.H["Subject"]) == 0 && pi.MI.Title != "" {
		pi.H["Subject"] = mail.OneHeaderVal(pi.MI.Title)
	}

	generateBody := func(
		w io.Writer, binary bool, poi mailib.PostObjectIndex) (err error) {

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

		_, err := io.Copy(w, r)
		if err != nil {
			err = fmt.Errorf("error copying: %v", err)
		}
		return
	}

	generateSomething := func(
		w io.Writer, binary bool, bo mailib.BodyObject) error {

		if poi, ok := bo.Data.(mailib.PostObjectIndex); ok {
			return generateBody(w, binary, poi)
		}
		if bo.Data == nil {
			return nil
		}
		panic("bad bo.Data type")
	}

	var generateMultipart func(
		w io.Writer, boundary string, pis []mailib.PartInfo) (err error)

	generateMultipart = func(
		w io.Writer, boundary string, pis []mailib.PartInfo) (err error) {

		pw := mail.NewPartWriter(w, boundary, "")
		for i := range pis {
			ppis, ismp := pis[i].Body.Data.([]mailib.PartInfo)
			var pb string
			if !ismp {
				if pis[i].ContentType != "" {
					pis[i].Headers["Content-Type"] =
						mail.OneHeaderVal(pis[i].ContentType)
				}
				// XXX should we announce 8bit text?
				if pis[i].Binary {
					pis[i].Headers["Content-Transfer-Encoding"] =
						mail.OneHeaderVal("base64")
				}
			} else {
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
				return generateSomething(w, pis[i].Binary, pis[i].Body)
			} else {
				return generateMultipart(w, pb, ppis)
			}
		}
		return pw.FinishParts("")
	}

	pis, ismp := pi.L.Body.Data.([]mailib.PartInfo)
	var bnd string
	if !ismp {
		// XXX should we announce 8bit text?
		if pi.L.Binary {
			pi.H["Content-Transfer-Encoding"] = mail.OneHeaderVal("base64")
		}
	} else {
		if len(pi.H["Content-Type"]) == 0 {
			return errNoContentType
		}
		pi.H["Content-Type"] = pi.H["Content-Type"][:1]
		pi.H["Content-Type"][0].V, bnd, err =
			modifyMultipartType(pi.H["Content-Type"][0].V)
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
