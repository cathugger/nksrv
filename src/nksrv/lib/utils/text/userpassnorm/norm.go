package userpassnorm

import (
	"errors"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	errEmptyString        = errors.New("empty string is not valid")
	errInvalidUTF8        = errors.New("invalid UTF-8")
	errUserCharNotAllowed = errors.New("only printable US-ASCII characters are allowed in username")
	errPassCharNotAllowed = errors.New("space and control characters are forbidden in passwords")
)

func NormaliseUser(s string) (_ string, err error) {
	// as admin I prefer something readable in the most shitty systems
	// actually this should be limited even more just haven't decided to what
	for i := 0; i < len(s); i++ {
		if s[i] <= 0x20 || s[i] >= 0x7F {
			err = errUserCharNotAllowed
			return
		}
	}
	if len(s) == 0 {
		err = errEmptyString
		return
	}
	return s, nil
}

func NormalisePass(s string) (_ string, err error) {
	if !utf8.ValidString(s) {
		err = errInvalidUTF8
		return
	}
	s = norm.NFC.String(s)
	for _, r := range s {
		if r <= 0x20 || r == 0x7F ||
			unicode.IsSpace(r) || !unicode.IsGraphic(r) {

			err = errPassCharNotAllowed
			return
		}
	}
	if len(s) == 0 {
		err = errEmptyString
		return
	}
	return s, nil
}
