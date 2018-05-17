package bufreader

import (
	"bytes"
	"errors"
	"io"
)

var ErrDelimNotFound = errors.New("bufreader: delimiter not found")

const defaultBufSize = 4096

type BufReader struct {
	u    io.Reader
	b    []byte
	w, r int
	err  error
}

func NewBufReader(u io.Reader) *BufReader {
	return &BufReader{u: u, b: make([]byte, defaultBufSize)}
}

func NewBufReaderSize(u io.Reader, s int) *BufReader {
	return &BufReader{u: u, b: make([]byte, s)}
}

func (r *BufReader) readErr() (err error) {
	err = r.err
	r.err = nil
	return
}

func (r *BufReader) Read(p []byte) (n int, err error) {
	if r.r == r.w {
		if r.err != nil {
			return 0, r.readErr()
		}
		if len(p) >= len(r.b) {
			// direct read
			return r.u.Read(p)
		}
		n, r.err = r.u.Read(r.b)
		if n <= 0 {
			return n, r.readErr()
		}
		r.r = 0
		r.w = n
	}
	n = copy(p, r.b[r.r:r.w])
	r.r += n
	return
}

func (r *BufReader) ReadUntil(p []byte, q byte) (n int, err error) {
	var x int
	for {
		if r.r == r.w {
			if r.err != nil {
				return n, r.readErr()
			}
			x, r.err = r.u.Read(r.b)
			if x <= 0 {
				return n, r.readErr()
			}
			r.r = 0
			r.w = x
		}
		x = r.w
		// clamp available source data to available destination space
		// so that we don't waste time scanning bytes we won't be able to write
		if r.w-r.r > len(p)-n {
			x = r.r + len(p) - n
		}
		// do search
		if i := bytes.IndexByte(r.b[r.r:x], q); i >= 0 {
			// copy will fully succeed because of previous clamp
			x = copy(p[n:], r.b[r.r:r.r+i+1])
			n += x
			r.r += x
			return
		} else {
			// copy ammount we can. this will fully succeed here too
			x = copy(p[n:], r.b[r.r:x])
			n += x
			r.r += x
		}
		if n >= len(p) {
			return n, ErrDelimNotFound
		}
	}
}
