package mail

import (
	"errors"
	"io"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/bufreader"
)

type ArticleReader interface {
	io.Reader
	ReadByte() (byte, error)
	Discard(n int) (int, error)
}

func ValidHeader(h []byte) bool {
	return au.IsPrintableASCIISlice(h, ':')
}

func estimateNumHeaders(br *bufreader.BufReader) (n int, e error) {
	br.CompactBuffer()
	_, e = br.FillBufferUpto(0)
	b := br.Buffered()
	cont := 0 // cont -- spare addition incase header line doesn't end with '\n'
	for i, c := range b {
		if c == '\n' {
			if cont == 0 {
				// \n\n or \n without any previous content -- end of headers
				return
			}
			if i+1 < len(b) && (b[i+1] == ' ' || b[i+1] == '\t') {
				// that's just continuation of previous header
				continue
			}
			n++
			cont = 0
		} else {
			cont = 1
		}
	}
	n += cont
	return
}

var (
	errTooLongHeader       = errors.New("too long header")
	errMissingColon        = errors.New("missing colon in header")
	errEmptyHeaderName     = errors.New("empty header name")
	errInvalidContinuation = errors.New("invalid header continuation")
)

const maxCommonHdrLen = 32

// common email headers statically allocated to avoid dynamic allocations
// TODO actually analyse which are used and update accordingly
var commonHeaders = map[string]string{
	// overrides
	// RFCs digestion
	"Message-Id":   "Message-ID",
	"Content-Id":   "Content-ID",
	"Mime-Version": "MIME-Version",
	// overchan
	"X-Pubkey-Ed25519":           "X-PubKey-Ed25519",
	"X-Signature-Ed25519-Sha512": "X-Signature-Ed25519-SHA512",
	"X-Frontend-Pubkey":          "X-Frontend-PubKey", // signature below
	"X-Encrypted-Ip":             "X-Encrypted-IP",
	"X-I2p-Desthash":             "X-I2P-Desthash",
}

func init() {
	// self-map overrides
	for _, v := range commonHeaders {
		commonHeaders[v] = v
	}
	// common headers which match their canonical versions
	for _, v := range [...]string{
		// kitchen-sink RFCs and other online sources digestion
		"Also-Control",
		"Approved",
		"Archive",
		"Bcc",
		"Bytes",
		"Cc",
		"Comments",
		"Content-Description",
		"Content-Disposition",
		"Content-Language",
		"Content-Transfer-Encoding",
		"Content-Type",
		"Control",
		"Date",
		"Distribution",
		"Expires",
		"Face",
		"Followup-To",
		"From",
		"Injection-Date",
		"Injection-Info",
		"Keywords",
		"Lines",
		"Newsgroups",
		"Organization",
		"Path",
		"Posting-Version",
		"Received",
		"References",
		"Relay-Version",
		"Return-Path",
		"Reply-To",
		"Sender",
		"Subject",
		"Summary",
		"Supersedes",
		"To",
		"In-Reply-To",
		"User-Agent",
		"Xref",
		"X-Antivirus",
		"X-Antivirus-Status",
		"X-Complaints-To",
		"X-Face",
		"X-Mailer",
		"X-Mozilla-News-Host",
		"X-Newsreader",
		"X-Trace",
		// overchan
		"X-Frontend-Signature", // pubkey above
		"X-Tor-Poster",
		"X-Sage",
	} {
		commonHeaders[v] = v
	}
}

type HeaderVal = string
type Headers map[string][]HeaderVal

func (h Headers) GetFirst(x string) HeaderVal {
	if s, ok := h[x]; ok {
		return s[0]
	}
	return ""
}

func (h Headers) Lookup(x string) []HeaderVal {
	if y, ok := commonHeaders[x]; ok {
		return h[y]
	}
	if s, ok := h[x]; ok {
		return s
	}

	var bx [maxCommonHdrLen]byte
	var b []byte
	if len(x) <= maxCommonHdrLen {
		b = bx[:len(x)]
	} else {
		b = make([]byte, len(x))
	}

	upper := true
	for i := 0; i < len(x); i++ {
		c := x[i]
		if upper && c >= 'a' && c <= 'z' {
			c = c - ('a' - 'A')
		}
		if !upper && c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		b[i] = c
		upper = c == '-'
	}
	// dont use commonHeaders there as there's no difference
	return h[string(b)]
}

func FindCommonCanonicalForm(s string) string {
	if len(s) > maxCommonHdrLen {
		return "" // not common
	}

	if y, ok := commonHeaders[s]; ok {
		return y
	}

	var b [maxCommonHdrLen]byte
	upper := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if upper && c >= 'a' && c <= 'z' {
			c = c - ('a' - 'A')
		}
		if !upper && c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		b[i] = c
		upper = c == '-'
	}
	return commonHeaders[string(b[:len(s)])]
}

// XXX can modify underlying storage
func mapCanonicalHeader(b []byte) string {
	// fast path: maybe its common header in form we want
	if h, ok := commonHeaders[string(b)]; ok {
		return h
	}
	// canonicalise
	upper := true
	for i, c := range b {
		if upper && c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
		if !upper && c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
		upper = c == '-'
	}
	// try to use static name again
	if h, ok := commonHeaders[string(b)]; ok {
		return h
	}
	// ohwell nothing we can do, just copy
	return string(b)
}
