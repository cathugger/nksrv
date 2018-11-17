package mail

import (
	"fmt"
	"io"
	"mime"
)

type failCharsetError string

func (f failCharsetError) Error() string {
	return fmt.Sprintf("unhandled charset %q", string(f))
}

func failCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	// failing this way if faster than fmt.Errorf done by default
	return nil, failCharsetError(charset)
}

var mimeWordDecoder = mime.WordDecoder{CharsetReader: failCharsetReader}

func DecodeMIMEWordHeader(s string) (string, error) {
	return mimeWordDecoder.DecodeHeader(s)
}
