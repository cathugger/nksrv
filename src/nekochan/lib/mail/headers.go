package mail

import (
	"errors"
	"io"

	au "nekochan/lib/asciiutils"
)

type ArticleReader interface {
	io.Reader
	ReadByte() (byte, error)
	Discard(n int) (int, error)
}

func ValidHeader(h []byte) bool {
	return au.IsPrintableASCIISlice(h, ':')
}

var (
	errTooLongHeader       = errors.New("too long header")
	errMissingColon        = errors.New("missing colon in header")
	errEmptyHeaderName     = errors.New("empty header name")
	errInvalidContinuation = errors.New("invalid header continuation")
)

const maxCommonHdrLen = 32

type HeaderVal = string
type Headers map[string][]HeaderVal

// case-sensitive
func (h Headers) GetFirst(x string) HeaderVal {
	if s, ok := h[x]; ok {
		// assumption: will always have at least one value
		return s[0]
	}
	return ""
}

// case-insensitive lookup
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
	if y, ok := commonHeaders[string(b)]; ok {
		return h[y]
	} else {
		return h[string(b)]
	}
}
