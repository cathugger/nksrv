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
	"X-Sage",
	// addressing
	"From",
	"Sender",
	"Newsgroups",
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

func writeHeaderLine(w io.Writer, h, s string, force bool) error {
	// TODO implement line folding
	if !force && len(h)+2+len(s) > 998 {
		return ErrHeaderLineTooLong
	}
	if _, e := fmt.Fprintf(w, "%s: %s\n", h, s); e != nil {
		return e
	}
	return nil
}

func writeHeaderLines(w io.Writer, h string, v []string, force bool) error {
	for _, s := range v {
		if e := writeHeaderLine(w, h, s, force); e != nil {
			return e
		}
	}
	return nil
}

func WriteHeaders(w io.Writer, H Headers, force bool) (err error) {
	// first try to put headers we know about in order
	for _, x := range writeHeaderOrder {
		if len(H[x]) != 0 {
			err = writeHeaderLines(w, x, H[x], force)
			if err != nil {
				return
			}
		}
	}
	// then try to put others in whatever order
	for h, v := range H {
		if _, inmap := writeHeaderMap[h]; !inmap {
			err = writeHeaderLines(w, h, v, force)
			if err != nil {
				return
			}
		}
	}
	// done
	return
}
