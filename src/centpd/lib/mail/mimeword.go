package mail

import (
	"errors"
	"fmt"
	"io"
	"mime"
	gmail "net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/ianaindex"
)

type failCharsetError string

func (e failCharsetError) Error() string {
	return fmt.Sprintf("unhandled charset %q", string(e))
}

func insaneCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	if charset == "" {
		return nil, failCharsetError("")
	}
	cod, e := ianaindex.MIME.Encoding(charset)
	if e != nil {
		return nil, e
	}
	return cod.NewDecoder().Reader(input), nil
}

var mimeWordDecoder = mime.WordDecoder{CharsetReader: insaneCharsetReader}

// DecodeMIMEWordHeader decodes MIME header and
// ensures result is valid UTF-8 text and does not contain null bytes
func DecodeMIMEWordHeader(s string) (_ string, err error) {
	s, err = mimeWordDecoder.DecodeHeader(s)
	if err != nil {
		return
	}
	// you aint gonna cheat me
	if !utf8.ValidString(s) {
		return "", errors.New("decoded string is invalid UTF-8")
	}
	// in most cases this is really invalid
	if strings.IndexByte(s, 0) >= 0 {
		return "", errors.New("decoded string contains null character")
	}
	// all ok
	return s, nil
}

var addrParser = gmail.AddressParser{WordDecoder: &mimeWordDecoder}

func ParseAddressX(s string) (a *gmail.Address, err error) {
	a, err = addrParser.Parse(s)
	if err != nil {
		i := strings.IndexByte(s, '<')
		if i >= 0 {
			j := strings.IndexByte(s[i+1:], '>')
			if j > 0 {
				// tolerate non-compliant messages from some older nntpchan nodes
				a = &gmail.Address{
					Name:    strings.TrimSpace(s[:i]),
					Address: s[i+1 : i+1+j],
				}
				err = nil
			}
		}
	}
	return
}

func isVchar(r rune) bool {
	// RFC 5234 B.1: VCHAR =  %x21-7E ; visible (printing) characters
	// RFC 6532 3.2: VCHAR =/ UTF8-non-ascii
	return (r >= 0x21 && r <= 0x7E) || r >= 0x80
}

func isAtext(r rune) bool {
	// RFC 5322: Printable US-ASCII characters not including specials.  Used for atoms.
	switch r {
	case '(', ')', '<', '>', '[', ']', ':', ';', '@', '\\', ',', '.', '"':
		return false
	}
	return isVchar(r)
}

func isWSP(r rune) bool { return r == ' ' || r == '\t' }

func isQtext(r rune) bool {
	if r == '\\' || r == '"' {
		return false
	}
	return isVchar(r)
}

func writeQuoted(b *strings.Builder, s string) {
	last := 0
	b.WriteByte('"')
	for i, r := range s {
		if !isQtext(r) && !isWSP(r) {
			if i > last {
				b.WriteString(s[last:i])
			}
			b.WriteByte('\\')
			b.WriteRune(r)
			last = i + utf8.RuneLen(r)
		}
	}
	if last < len(s) {
		b.WriteString(s[last:])
	}
	b.WriteByte('"')
}

func FormatAddress(name, email string) string {
	// somewhat based on stdlib' mail.Address.String()

	b := &strings.Builder{}

	if name != "" {
		needsEncoding := false
		needsQuoting := false
		for _, r := range name {
			if r >= 0x80 || (!isWSP(r) && !isVchar(r)) {
				needsEncoding = true
				break
			}
			if !isAtext(r) {
				needsQuoting = true
			}
		}

		if needsEncoding {
			// Text in an encoded-word in a display-name must not contain certain
			// characters like quotes or parentheses (see RFC 2047 section 5.3).
			// When this is the case encode the name using base64 encoding.
			if strings.ContainsAny(name, "\"#$%&'(),.:;<>@[]^`{|}~") {
				b.WriteString(mime.BEncoding.Encode("utf-8", name))
			} else {
				b.WriteString(mime.QEncoding.Encode("utf-8", name))
			}
		} else if needsQuoting {
			writeQuoted(b, name)
		} else {
			b.WriteString(name)
		}

		b.WriteByte(' ')
	}

	at := strings.LastIndex(email, "@")
	var local, domain string
	if at >= 0 {
		local, domain = email[:at], email[at+1:]
	} else {
		local = email
	}

	quoteLocal := false
	for i, r := range local {
		if isAtext(r) {
			// if atom then okay
			continue
		}
		if r == '.' && r > 0 && local[i-1] != '.' && i < len(local)-1 {
			// dots are okay but only if surrounded by non-dots
			continue
		}
		quoteLocal = true
		break
	}

	b.WriteByte('<')
	if !quoteLocal {
		b.WriteString(local)
	} else {
		writeQuoted(b, local)
	}
	b.WriteByte('@')
	b.WriteString(domain)
	b.WriteByte('>')

	return b.String()
}
