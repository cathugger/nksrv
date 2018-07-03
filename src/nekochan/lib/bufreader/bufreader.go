package bufreader

import (
	"bytes"
	"errors"
	"io"
)

var ErrDelimNotFound = errors.New("bufreader: delimiter not found")
var errInvalidUnread = errors.New("bufreader: invalid use of UnreadByte")

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
	if s <= 0 {
		panic("size must be >0\n")
	}
	return &BufReader{u: u, b: make([]byte, s)}
}

func (r *BufReader) SetReader(u io.Reader) {
	r.u = u
}

func (r *BufReader) ResetErr() {
	r.err = nil
}

func (r *BufReader) QueuedErr() error {
	return r.err
}

func (r *BufReader) readErr() (err error) {
	err = r.err
	r.err = nil
	return
}

// implements io.Reader interface
func (r *BufReader) Read(p []byte) (n int, _ error) {
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
			return 0, r.readErr()
		}
		r.r = 0
		r.w = n
	}
	n = copy(p, r.b[r.r:r.w])
	r.r += n
	return
}

func (r *BufReader) ReadByte() (byte, error) {
	if r.r == r.w {
		if r.err != nil {
			return 0, r.readErr()
		}
		var n int
		n, r.err = r.u.Read(r.b)
		if n <= 0 {
			return 0, r.readErr()
		}
		r.r = 0
		r.w = n
	}
	c := r.b[r.r]
	r.r++
	return c, nil
}

func (r *BufReader) UnreadByte(c byte) error {
	if r.r != 0 {
		r.r--
		r.b[r.r] = c
	} else {
		if r.w == 0 {
			r.w = 1
			r.b[0] = c
		} else {
			return errInvalidUnread
		}
	}
	return nil
}

// reads into buffer supplied in p parameter until byte supplied in q parameter is found.
// filled buffer contains last byte specified as q.
// returns number of bytes written into p, and error, either generic or in case q was not found and p was filled.
func (r *BufReader) ReadUntil(p []byte, q byte) (n int, _ error) {
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

func (r *BufReader) Size() int {
	return len(r.b)
}

func (r *BufReader) Buffered() []byte {
	return r.b[r.r:r.w]
}

func (r *BufReader) Capacity() int {
	if r.r == r.w {
		r.r = 0
		r.w = 0
	}
	return len(r.b) - r.r
}

func (r *BufReader) CompactBuffer() {
	if r.r != 0 {
		copy(r.b, r.b[r.r:r.w])
		r.w -= r.r
		r.r = 0
	}
}

func (r *BufReader) FillBufferAtleast(w int) (n int, _ error) {
	if r.r == r.w {
		r.r = 0
		r.w = 0
	}
	var x int
	for r.w < len(r.b) && n < w {
		if r.err != nil {
			return n, r.readErr()
		}
		x, r.err = r.u.Read(r.b[r.w:])
		if x <= 0 {
			return n, r.readErr()
		}
		r.w += x
		n += x
	}
	return
}

func (r *BufReader) FillBufferUpto(w int) (n int, _ error) {
	if r.r == r.w {
		r.r = 0
		r.w = 0
	}
	var x int
	for r.w < len(r.b) && (w <= 0 || r.w-r.r < w) {
		if r.err != nil {
			if w <= 0 && n != 0 {
				break
			}
			return n, r.readErr()
		}
		x, r.err = r.u.Read(r.b[r.w:])
		if x <= 0 {
			return n, r.readErr()
		}
		r.w += x
		n += x
	}
	return
}

// skips specified ammount of bytes. if specified ammount is negative, read until fail.
// returns skipped ammount of bytes and error if specified ammount could not be skipped.
func (r *BufReader) Discard(n int) (s int, _ error) {
	var x int
	for {
		if n >= 0 && r.w-r.r >= n {
			// existing buffer is enough to satisfy
			r.r += n
			s += n
			return s, nil
		}
		// existing buffer is too small to satisfy so just eat it whole
		if n > 0 {
			n -= r.w - r.r
		}
		s += r.w - r.r
		r.Drop()

		if r.err != nil {
			return s, r.readErr()
		}

		x, r.err = r.u.Read(r.b)
		if x <= 0 {
			return s, r.readErr()
		}
		r.r = 0
		r.w = x
	}
}

// discards all cached data. use only if you know what you are doing.
func (r *BufReader) Drop() {
	r.r = 0
	r.w = 0
}
