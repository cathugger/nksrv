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

func ParseAddressX(s string) (*gmail.Address, error) {
	return addrParser.Parse(s)
}
