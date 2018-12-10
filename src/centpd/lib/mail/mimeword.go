package mail

import (
	"errors"
	"fmt"
	"io"
	"mime"
	gmail "net/mail"
	"strings"
	"unicode/utf8"
)

type failCharsetError string

func (f failCharsetError) Error() string {
	return fmt.Sprintf("unhandled charset %q", string(f))
}

func failCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	// failing this way is faster than fmt.Errorf done by default
	return nil, failCharsetError(charset)
}

var mimeWordDecoder = mime.WordDecoder{CharsetReader: failCharsetReader}

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

func ParseAddressX(s string) (*gmail.Address, error) {
	return gmail.ParseAddress(s)
}
