package tmplrenderer

import (
	"io"
	"net/http"
)

type nopWCloser struct {
	io.Writer
}

func (nopWCloser) Close() error {
	return nil
}

func nopWCCreator(w http.ResponseWriter) io.WriteCloser {
	return nopWCloser{w}
}

var _ wcCreator = nopWCCreator
