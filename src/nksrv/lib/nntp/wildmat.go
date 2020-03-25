package nntp

import (
	"bytes"
	"regexp"
	"unicode/utf8"
)

func ValidWildmat(x []byte) bool {
	/*
	 * {RFC 3977}
	 * wildmat = wildmat-pattern *("," ["!"] wildmat-pattern)
	 * wildmat-pattern = 1*wildmat-item
	 * wildmat-item = wildmat-exact / wildmat-wild
	 * wildmat-exact = %x22-29 / %x2B / %x2D-3E / %x40-5A / %x5E-7E /
	 *   UTF8-non-ascii ; exclude ! * , ? [ \ ]
	 * wildmat-wild = "*" / "?"
	 */
	const (
		sStartPattern = iota
		sInsidePattern
		sNegate
	)
	hasunicode := false
	s := sStartPattern
	for _, c := range x {
		if c >= 0x80 {
			hasunicode = true
		}
		if (c >= 0x22 && c <= 0x29) || c == 0x2B ||
			(c >= 0x2D && c <= 0x3E) || (c >= 0x40 && c <= 0x5A) ||
			(c >= 0x5E && c <= 0x7E) || c >= 0x80 /* wildmat-exact */ ||
			c == '*' || c == '?' /* wildmat-wild */ {
			s = sInsidePattern
			continue
		}
		// '!' only allowed in front of pattern
		if c == '!' && s == sStartPattern {
			s = sNegate
			continue
		}
		if c == ',' && s == sInsidePattern {
			s = sStartPattern // next char must be start of new pattern or '!'
			continue
		}
		return false
	}
	// cannot end with ',' or '!'
	return s == sInsidePattern && (!hasunicode || utf8.Valid(x))
}

type wildmatPiece struct {
	re     *regexp.Regexp
	result bool
}

type Wildmat []wildmatPiece

func estimateTranslatedSize(x []byte) (max int) {
	var this int
	for _, c := range x {
		// [[:punct:]]    punctuation (== [!-/:-@[-`{-~])
		// ! " # $ % & ' ( ) * + , - . /
		// : ; < = > ? @
		// [ \ ] ^ _ `
		// { | } ~
		// basically all non-alnum printables (except space but that cant happen in wildmat)
		// \( \) \. \| \+  \^ \$ ......
		// easier to whitelist alnum
		// '*' -> ".*"
		// `"?" matches exactly one character (which may be more than one octet).` -> '.'
		if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '?' {
			this++
			continue
		}
		if c == ',' {
			if this > max {
				max = this
			}
			this = 0
			continue
		}
		// not included in regexp itself
		if c == '!' {
			continue
		}
		// punct char, to be escaped
		this += 2
	}
	if this > max {
		max = this
	}
	return
}

// TODO benchmark and possibly replace with stripped down https://github.com/danwakefield/fnmatch
func CompileWildmat(x []byte) (w Wildmat) {
	b := bytes.NewBuffer(make([]byte, 0, 2+estimateTranslatedSize(x)))

	result := true

	b.WriteByte('^')
	for _, c := range x {
		if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			// dont need special processing
			b.WriteByte(c)
			continue
		}
		if c == '?' {
			// '?' -> '.'
			b.WriteByte('.')
			continue
		}
		if c == '*' {
			// '*' -> ".*"
			b.WriteByte('.')
			b.WriteByte('*')
			continue
		}
		if c == ',' {
			b.WriteByte('$')
			w = append(w, wildmatPiece{
				re:     regexp.MustCompile(unsafeBytesToStr(b.Bytes())),
				result: result,
			})
			b.Reset()
			b.WriteByte('^')
			result = true
			continue
		}
		if c == '!' {
			result = false
			continue
		}
		// escape
		b.WriteByte('\\')
		b.WriteByte(c)
	}
	b.WriteByte('$')
	w = append(w, wildmatPiece{
		re:     regexp.MustCompile(unsafeBytesToStr(b.Bytes())),
		result: result,
	})
	return
}

func (w Wildmat) CheckString(s string) (result bool) {
	for i := range w {
		// later ones override previous ones
		if w[i].re.MatchString(s) {
			result = w[i].result
		}
	}
	return
}

func (w Wildmat) CheckBytes(s []byte) (result bool) {
	for i := range w {
		if w[i].re.Match(s) {
			result = w[i].result
		}
	}
	return
}
