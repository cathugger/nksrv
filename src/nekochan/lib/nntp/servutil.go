package nntp

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func parseKeyword(b []byte) int {
	i := 0
	l := len(b)
	for {
		if i >= l {
			return i
		}
		c := b[i]
		if c == ' ' || c == '\t' {
			return i
		}
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
		i++
	}
}

func ToUpperASCII(b []byte) {
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
}

func ToLowerASCII(b []byte) {
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
}

// NOTE ASCII space (32) is neither printable chatacter nor control character
func isPrintableASCIISlice(s []byte, e byte) bool {
	for _, c := range s {
		if c <= 32 || c >= 127 || c == e {
			return false
		}
	}
	return true
}

func CutMessageID(id FullMsgID) CoreMsgID {
	return CoreMsgID(id[1 : len(id)-1])
}

func ValidMessageID(id FullMsgID) bool {
	return len(id) >= 3 && id[0] == '<' && id[len(id)-1] == '>' &&
		len(id) <= 250 && isPrintableASCIISlice(CutMessageID(id), '>')
}

func ReservedMessageID(id FullMsgID) bool {
	sid := unsafeBytesToStr(id)
	return sid == "<0>" /* {RFC 977} */ ||
		sid == "<keepalive@dummy.tld>" /* srndv2 */
}

func validMessageNum(n uint64) bool {
	return int64(n) > 0
}

func validHeaderQuery(hq []byte) bool {
	if hq[0] == ':' {
		hq = hq[1:]
	}
	return isPrintableASCIISlice(hq, ':')
}

func validHeader(h []byte) bool {
	return isPrintableASCIISlice(h, ':')
}

func ValidGroupSlice(s []byte) bool {
	for _, c := range s {
		if !((c >= 0x22 && c <= 0x29) || c == 0x2B ||
			(c >= 0x2D && c <= 0x3E) || (c >= 0x40 && c <= 0x5A) ||
			(c >= 0x5E && c <= 0x7E) || c >= 0x80) {
			return false
		}
	}
	return true
}

func FullValidGroupSlice(s []byte) bool {
	hasunicode := false
	for _, c := range s {
		if !((c >= 0x22 && c <= 0x29) || c == 0x2B ||
			(c >= 0x2D && c <= 0x3E) || (c >= 0x40 && c <= 0x5A) ||
			(c >= 0x5E && c <= 0x7E) || c >= 0x80) {
			return false
		}
		if c >= 0x80 {
			hasunicode = true
		}
	}
	return !hasunicode || utf8.Valid(s)
}

func parseRange(srange string) (rmin, rmax int64, valid bool) {
	rmin = 1
	rmax = -1
	// [num[-[num]]]
	if i := strings.IndexByte(srange, '-'); i >= 0 {
		if i != 0 {
			n, e := strconv.ParseUint(srange[:i], 10, 64)
			if e != nil {
				return rmin, rmax, false
			}
			if int64(n) >= 0 {
				rmin = int64(n)
			}
		}
		if i+1 != len(srange) {
			n, e := strconv.ParseUint(srange[i+1:], 10, 64)
			if e != nil {
				return rmin, rmax, false
			}
			if int64(n) >= 0 {
				rmax = int64(n)
			}
		}
	} else {
		n, e := strconv.ParseUint(srange, 10, 64)
		if e != nil {
			return rmin, rmax, false
		}
		rmin = int64(n)
		rmax = rmin
	}
	return rmin, rmax, true
}

func isNumberSlice(x []byte) bool {
	for _, c := range x {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func stoi(x []byte) (n int) {
	for _, c := range x {
		n = n*10 + int(c)
	}
	return
}

func parseDateSlice(date []byte) (Y, M, D int, valid bool) {
	if len(date) < 5 || len(date) > 12 || !isNumberSlice(date) {
		return Y, M, D, false
	}

	D = int(date[len(date)-2])*10 + int(date[len(date)-1])
	M = int(date[len(date)-4])*10 + int(date[len(date)-3])
	if len(date) != 6 {
		Y = stoi(date[:len(date)-4])
	} else {
		Y = int(date[0])*10 + int(date[1])
		/*
		 * {RFC 3977}
		 * If the first two digits of the year are not specified
		 * (this is supported only for backward compatibility), the year is to
		 * be taken from the current century if yy is smaller than or equal to
		 * the current year, and the previous century otherwise.
		 */
		CY := time.Now().UTC().Year()
		CYa, CYb := CY/100, CY%100
		if Y <= CYb {
			Y += CYa * 100
		} else {
			Y += (CYa - 1) * 100
		}
	}
	return Y, M, D, M > 0 && M <= 12 && D > 0
}

func parseTimeSlice(t []byte) (h, m, s int, valid bool) {
	if len(t) != 6 || !isNumberSlice(t) {
		return h, m, s, false
	}
	h = int(t[0])*10 + int(t[1])
	m = int(t[2])*10 + int(t[3])
	s = int(t[4])*10 + int(t[5])
	return h, m, s, h <= 24 && m < 60
}

func parseDateTime(w Responder, ds, ts []byte) (t time.Time, v bool) {
	var Y, M, D, h, m, s int

	if Y, M, D, v = parseDateSlice(ds); !v {
		w.PrintfLine("501 invalid date")
		return
	}

	if h, m, s, v = parseTimeSlice(ts); !v {
		w.PrintfLine("501 invalid time")
		return
	}

	t = time.Date(Y, time.Month(M), D, h, m, s, 0, time.UTC)
	return
}
