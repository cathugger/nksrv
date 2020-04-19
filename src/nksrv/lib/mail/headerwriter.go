package mail

import (
	"errors"
	"fmt"
	"io"
)

var ErrHeaderLineTooLong = errors.New("header line is too long")

var writeHeaderOrder = [...]string{
	// main logic things
	"Message-ID",
	"Subject",
	"Date",
	"Control",
	"Also-Control",
	"X-Sage",
	// addressing
	"From",
	"Sender",
	"Reply-To",
	"Newsgroups",
	"Followup-To",
	"To",
	"Cc",
	"Bcc",
	"References",
	"In-Reply-To",
	"Distribution",
	"Expires",
	"Supersedes",
	"Approved",
	"Organization",
	"Keywords",
	"Summary",
	"Comments",
	// info about network
	"Path",
	// info from injection node
	"Injection-Date",
	"Injection-Info",
	"NNTP-Posting-Date",
	"NNTP-Posting-Host",
	"X-Frontend-PubKey",
	"X-Frontend-Signature",
	"X-Encrypted-IP",
	"X-Tor-Poster",
	"X-I2P-DestHash",
	"X-Antivirus",
	"X-Antivirus-Status",
	// info from client
	"User-Agent",
	"X-Mailer",
	"X-Newsreader",
	"X-Mozilla-News-Host",
	// info about content
	"X-PubKey-Ed25519",
	"X-Signature-Ed25519-SHA512",
	"Cancel-Key",
	"Cancel-Lock",
	"MIME-Version",
	"Content-Type",
	"Content-Transfer-Encoding",
	"Content-Disposition",
	"Content-Description",
	"Content-Language",
	"Bytes",
	"Lines",
}

// mask map of above
var writeHeaderMap = func() (m map[string]struct{}) {
	m = make(map[string]struct{})
	for _, x := range writeHeaderOrder {
		m[x] = struct{}{}
	}
	return
}()

var writePartHeaderOrder = [...]string{
	"Content-ID",
	"Content-Type",
	"Content-Transfer-Encoding",
	"Content-Disposition",
	"Content-Description",
	"Content-Language",
}

// mask map of above
var writePartHeaderMap = func() (m map[string]struct{}) {
	m = make(map[string]struct{})
	for _, x := range writePartHeaderOrder {
		m[x] = struct{}{}
	}
	return
}()

func writeHeaderLine(w io.Writer, h, s string, force bool) error {
	// TODO implement line folding
	if !force && len(h)+2+len(s)+2 > maxHeaderLen {
		return ErrHeaderLineTooLong
	}
	if _, e := fmt.Fprintf(w, "%s: %s\n", h, s); e != nil {
		return e
	}
	return nil
}

func writeHeaderLines(w io.Writer, h string, v []HeaderMapVal, force bool) error {
	for _, x := range v {
		hh := h
		if x.O != "" {
			hh = x.O
		}
		if e := writeHeaderLine(w, hh, x.V, force); e != nil {
			return e
		}
	}
	return nil
}

func whlFunc(w io.Writer) addHdrFunc {
	return func(h string, hmvl []HeaderMapVal, force bool) error {
		return writeHeaderLines(w, h, hmvl, force)
	}
}

func WriteMessageHeaderMap(w io.Writer, H HeaderMap, force bool) error {
	return addHeadersOrdered(
		whlFunc(w), writeHeaderOrder[:], writeHeaderMap, H, force)
}

func WritePartHeaderMap(w io.Writer, H HeaderMap, force bool) error {
	return addHeadersOrdered(
		whlFunc(w), writePartHeaderOrder[:], writePartHeaderMap, H, force)
}

func WriteHeaderList(w io.Writer, HL HeaderList, force bool) (err error) {
	for _, x := range HL {
		hh := x.K
		if x.O != "" {
			hh = x.O
		}
		if err = writeHeaderLine(w, hh, x.V, force); err != nil {
			return
		}
	}
	return
}
