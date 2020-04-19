package mail

import (
	"encoding/json"
	"errors"
	"sort"
	"unicode/utf8"

	au "nksrv/lib/asciiutils"
)

/*
 * some utility stuff
 */
func ValidHeaderName(h []byte) bool {
	return au.IsPrintableASCIISlice(h, ':')
}

func validHeaderContent(b []byte) bool {
	has8bit := false
	for _, c := range b {
		if c == '\000' || c == '\r' || c == '\n' {
			return false
		}
		if c&0x80 != 0 {
			has8bit = true
		}
	}
	return !has8bit || utf8.Valid(b)
}

const maxHeaderLen = 2000

var (
	errTooLongHeader       = errors.New("too long header")
	errMissingColon        = errors.New("missing colon in header")
	errEmptyHeaderName     = errors.New("empty header name")
	errInvalidContinuation = errors.New("invalid header continuation")
)

const maxCommonHdrLen = 32

/*
 * header map stuff
 */

type HeaderValSplit = uint32
type HeaderValSplitList = []HeaderValSplit

type HeaderMapValInner struct {
	V string             `json:"v"`           // value
	O string             `json:"h,omitempty"` // original name, optional, needed only incase non-canonical form
	S HeaderValSplitList `json:"s,omitempty"` // split points, for folding/unfolding
}

type HeaderMapVal struct {
	HeaderMapValInner
}

type HeaderMapVals []HeaderMapVal
type HeaderMap map[string]HeaderMapVals

// header map related functions

func (hv HeaderMapVal) MarshalJSON() ([]byte, error) {
	if hv.O == "" && len(hv.S) == 0 {
		return json.Marshal(hv.V)
	} else {
		return json.Marshal(hv.HeaderMapValInner)
	}
}

func (hv *HeaderMapVal) UnmarshalJSON(b []byte) (err error) {
	err = json.Unmarshal(b, &hv.V)
	if err == nil {
		hv.O = ""
		hv.S = []uint32(nil)
		return
	}
	return json.Unmarshal(b, &hv.HeaderMapValInner)
}

func OneHeaderVal(v string) HeaderMapVals {
	return HeaderMapVals{{HeaderMapValInner: HeaderMapValInner{V: v}}}
}

// case-sensitive
func (h HeaderMap) GetFirst(x string) string {
	if s, ok := h[x]; ok {
		// assumption: will always have at least one value
		return s[0].V
	}
	return ""
}

// case-insensitive lookup
// NOTE: assumes that HeaderMap was created by current version
// this is NOT the case with stuff pulled out of database
func (h HeaderMap) Lookup(x string) []HeaderMapVal {
	if y, ok := headerMap[x]; ok {
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
	if y, ok := headerMap[string(b)]; ok {
		return h[y]
	} else {
		return h[string(b)]
	}
}

/*
 * header list stuff
 */

type HeaderListValInner struct {
	K string             `json:"k"`
	V string             `json:"v"`
	O string             `json:"h"`
	S HeaderValSplitList `json:"s"`
}

type HeaderListVal struct {
	HeaderListValInner
}

type HeaderList []HeaderListVal

/*
 * header map related functions
 */

func (hv HeaderListVal) MarshalJSON() ([]byte, error) {
	if hv.O == "" && len(hv.S) == 0 {
		l := [2]string{
			hv.K,
			hv.V,
		}
		return json.Marshal(l)
	} else {
		return json.Marshal(hv.HeaderListValInner)
	}
}

func (hv *HeaderListVal) UnmarshalJSON(b []byte) (err error) {
	var l [2]string
	err = json.Unmarshal(b, &l)
	if err == nil {
		*hv = HeaderListVal{HeaderListValInner{
			K: l[0],
			V: l[1],
		}}
		return
	}
	return json.Unmarshal(b, &hv.HeaderListValInner)
}

func (hl HeaderList) ToHeaderMap() (hm HeaderMap) {

	// assume HeaderList is all canonical already

	// make independent list which is going to be used for this
	chml := make(HeaderMapVals, 0, len(hl))

	hm = make(HeaderMap)
	for _, hlv := range hl {
		hmv := HeaderMapVal{HeaderMapValInner{
			V: hlv.V,
			O: hlv.O,
			S: hlv.S,
		}}
		if lastval := hm[hlv.K]; lastval != nil {
			hm[hlv.K] = append(lastval, hmv)
		} else {
			chml = append(chml[len(chml):], hmv)
			hm[hlv.K] = chml[0:1:1]
		}
	}

	return
}

type addHdrFunc = func(h string, hmvl []HeaderMapVal, force bool) error

func addHeadersOrdered(
	F addHdrFunc, HO []string, HM map[string]struct{},
	H HeaderMap, force bool) (
	err error) {

	n := 0
	// first try to put headers we know about in order
	for _, h := range HO {
		if len(H[h]) != 0 {
			n++
			err = F(h, H[h], force)
			if err != nil {
				return
			}
		}
	}
	if len(H) <= n {
		return
	}
	// then put rest, sorted
	l := make([]string, 0, len(H)-n)
	for k := range H {
		if _, inmap := HM[k]; !inmap {
			l = append(l, k)
		}
	}
	sort.Strings(l)
	for _, h := range l {
		err = F(h, H[h], force)
		if err != nil {
			return
		}
	}
	// done
	return
}

func (hm HeaderMap) toHeaderList(
	HO []string, HM map[string]struct{}) (hl HeaderList) {

	f := func(h string, hmvl []HeaderMapVal, force bool) error {
		for _, hmv := range hmvl {
			hl = append(hl, HeaderListVal{HeaderListValInner{
				K: h,
				V: hmv.V,
				O: hmv.O,
				S: hmv.S,
			}})
		}
		return nil
	}
	_ = addHeadersOrdered(f, HO, HM, hm, false)
	return
}

func (hm HeaderMap) ToMessageHeaderList() HeaderList {
	return hm.toHeaderList(writeHeaderOrder[:], writeHeaderMap)
}
func (hm HeaderMap) ToPartHeaderList() (hl HeaderList) {
	return hm.toHeaderList(writePartHeaderOrder[:], writePartHeaderMap)
}
