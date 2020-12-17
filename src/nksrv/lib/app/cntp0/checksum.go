package cntp0

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	qp "mime/quotedprintable"

	"nksrv/lib/mail"
	au "nksrv/lib/utils/text/asciiutils"
)

var (
	identStart      = []byte("$CNTP0$")
	identClose      = []byte("$")
	digestSeparator = []byte("$")
	newline         = []byte("\n")
)

func writeIdentification(w io.Writer, d Digest) {
	w.Write(identStart)
	d.WriteIdentifier(w)
	w.Write(identClose)
}

var excHeadHeaders = map[string]struct{}{
	"Message-ID":                {},
	"Path":                      {},
	"Xref":                      {},
	"MIME-Version":              {},
	"Content-Type":              {},
	"Content-Transfer-Encoding": {},
	"X-CNTP-Headers":            {},
}

var excPartHeaders = map[string]struct{}{
	"Content-ID":                {},
	"Content-Type":              {},
	"Content-Disposition":       {},
	"Content-Transfer-Encoding": {},
	"X-CNTP-Headers":            {},
}

var (
	errMultipartEncoding = errors.New("wrong Content-Transfer-Encoding for multipart type")
	errNoBoundary        = errors.New("multipart has no boundary specified")
)

func MakeChecksum(d Digest, res io.Writer, m mail.MessageHead) error {
	hasher := d.Hasher()
	w := hasher.h

	// hash identification
	writeIdentification(w, d)
	w.Write(newline)

	// implicit header: X-CNTP-Headers
	hdrs := m.H.GetFirst("X-CNTP-Headers")
	w.Write(unsafeStrToBytes(hdrs))
	w.Write(newline)

	multipart := false

	// implicit header: Content-Type (even if empty)
	ct := au.TrimWSString(m.H.GetFirst("Content-Type"))
	ctp := au.TrimWSString(au.UntilString(ct, ';'))
	if au.StartsWithFoldString(ctp, "multipart/") {
		multipart = true
		w.Write(unsafeStrToBytes(ctp))
	} else {
		w.Write(unsafeStrToBytes(ct))
	}
	w.Write(newline)

	err = checksumHeadersAndBody(d, w, m.H, m.B, hdrs, excHeadHeaders)
	if err != nil {
		return err
	}

	var digest [MaxBytes]byte
	_, err := res.Write(w.Sum(digest[:0]))
	return err
}

func checksumPart(d Digest, res io.Writer, pr *mail.PartReader) error {
	hasher := d.Hasher()
	w := hasher.h

	H, err := pr.ReadHeaders(8 * 1024)
	if err != nil {
		return err
	}

	// implicit headers
	fmt.Fprintf(w, "%s\n", H.GetFirst("Content-ID"))

	var multipart bool
	ct := H.GetFirst("Content-Type")
	ctp := au.TrimWSString(au.UntilString(ct, ';'))
	if au.StartsWithFoldString(ctp, "multipart/") {
		multipart = true
		w.Write(unsafeStrToBytes(ctp))
	} else {
		multipart = false
		w.Write(unsafeStrToBytes(ct))
	}
	w.Write(newline)

	fmt.Fprintf(w, "%s\n", H.GetFirst("Content-Disposition"))

	hdrs := H.GetFirst("X-CNTP-Headers")

	fmt.Fprintf(w, "%s\n", hdrs)

	err = checksumHeadersAndBody(w, H, pr, hdrs, excPartHeaders)
	if err != nil {
		return err
	}

	var digest [MaxBytes]byte
	_, err = res.Write(w.Sum(digest[:0]))
	return err
}

// check explicit headers and body
func checksumHeadersAndBody(
	d Digest, w io.Writer, H mail.Headers, r io.Reader,
	hl string, hexc map[string]struct{}) error {

	au.IterateFields(hl, func(x string) {
		xx := m.H.Lookup(x)
		ll := len(xx)
		if c := mail.FindCommonCanonicalKey(x); c != "" {
			if _, ok := hexc[c]; ok {
				// skip this one
				ll = 0
			}
		}

		fmt.Fprintf(w, "%d\n", ll)
		for i := 0; i < ll; i++ {
			w.Write(unsafeStrToBytes(xx[i].V))
			w.Write(newline)
		}
	})

	cte := au.TrimWSString(m.H.GetFirst("Content-Transfer-Encoding"))
	var r io.Reader = m.B
	binary := false
	if cte != "" {
		if au.EqualFoldString(cte, "base64") {
			r = base64.NewDecoder(base64.StdEncoding, r)
			binary = true
			if multipart {
				// non-clear encodings not allowed, error
				return errMultipartEncoding
			}
		} else if au.EqualFoldString(cte, "quoted-printable") {
			r = qp.NewReader(r)
			if multipart {
				// non-clear encodings not allowed, error
				return errMultipartEncoding
			}
		}
		// else assume 7bit/8bit
	}
	if !multipart {
		if !binary {
			r = au.NewUnixTextReader(r)
		}
		if !binary {
			fmt.Fprintf(w, "text\n")
		} else {
			fmt.Fprintf(w, "binary\n")
		}
		_, e := io.Copy(w, r)
		if e != nil {
			return e
		}
	} else {
		var err error
		_, param, err := mime.ParseMediaType(ct)
		if err != nil {
			return err
		}
		if param["boundary"] == "" {
			return errNoBoundary
		}
		fmt.Fprintf(w, "multipart\n")
		pr := mail.NewPartReader(r, param["boundary"])
		for err = pr.NextPart(); err == nil; err = pr.NextPart() {
			err = checksumPart(d, w, pr)
			if err != nil {
				pr.Close()
				return err
			}
		}
		pr.Close()
		if err != io.EOF {
			return err
		}
	}
}
