package mail

import (
	"encoding/json"
	"errors"

	au "centpd/lib/asciiutils"
)

func ValidHeader(h []byte) bool {
	return au.IsPrintableASCIISlice(h, ':')
}

const maxHeaderLen = 2000

var (
	errTooLongHeader       = errors.New("too long header")
	errMissingColon        = errors.New("missing colon in header")
	errEmptyHeaderName     = errors.New("empty header name")
	errInvalidContinuation = errors.New("invalid header continuation")
)

const maxCommonHdrLen = 32

type HeaderValInner struct {
	V string   `json:"v"`           // value
	H string   `json:"h,omitempty"` // original name, optional, needed only incase non-canonical form
	S []uint32 `json:"s,omitempty"` // split points, for folding/unfolding
}

type HeaderVal struct {
	HeaderValInner
}

func (hv HeaderVal) MarshalJSON() ([]byte, error) {
	if hv.H == "" && len(hv.S) == 0 {
		return json.Marshal(hv.V)
	} else {
		return json.Marshal(hv.HeaderValInner)
	}
}

func (hv *HeaderVal) UnmarshalJSON(b []byte) (err error) {
	err = json.Unmarshal(b, &hv.V)
	if err == nil {
		hv.H = ""
		hv.S = []uint32(nil)
		return
	}
	return json.Unmarshal(b, &hv.HeaderValInner)
}

type HeaderVals []HeaderVal
type Headers map[string]HeaderVals

func OneHeaderVal(v string) HeaderVals {
	return HeaderVals{{HeaderValInner: HeaderValInner{V: v}}}
}

// case-sensitive
func (h Headers) GetFirst(x string) string {
	if s, ok := h[x]; ok {
		// assumption: will always have at least one value
		return s[0].V
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
