// XXX move to its own package?

package bufreader

import (
	"errors"
	"io"
)

type DotReader struct {
	r   *BufReader // underlying reader
	s   int        // parser state
	snl int        // strict newline processing
}

const (
	snlNone int = iota
	snlEnabled
	snlUnexpectedCR
	snlUnexpectedLF
)

var errUnexpectedCR = errors.New("dotreader: CR not part of CR-LF sequence")
var errUnexpectedLF = errors.New("dotreader: LF not part of CR-LF sequence")

func NewDotReader(r *BufReader, permissive bool) (dr *DotReader) {
	dr = &DotReader{r: r}
	if !permissive {
		dr.snl = snlEnabled
	}
	return
}

const (
	sBeginLine = iota // begin of line
	sDot              // dot after begin of line
	sNonBegin         // after initial char of line
	sDotCR            // dot CR
	sCR               // non-dot CR
	sEOF              // after CRLF.CRLF
)

func (d *DotReader) Reset() {
	d.s = 0
}

func (d *DotReader) process(c byte) (byte, bool) {
	switch d.s {
	case sBeginLine:
		if c == '.' {
			d.s = sDot
			// skip initial . regardless of stuff going after it
			return 0, false
		}
		if c == '\r' {
			d.s = sCR
			return 0, false
		}
		if c != '\n' {
			d.s = sNonBegin
		} else {
			// LF at the start of line
			if d.snl != 0 {
				d.snl = snlUnexpectedLF
				d.s = sEOF
				return 0, false
			}
			d.s = sBeginLine
		}

	case sDot:
		if c == '\r' {
			d.s = sDotCR
			return 0, false
		}
		if c == '\n' {
			// LF after dot
			if d.snl != 0 {
				d.snl = snlUnexpectedLF
			}
			d.s = sEOF
			return 0, false
		}
		d.s = sNonBegin

	case sNonBegin:
		if c == '\r' {
			d.s = sCR
			return 0, false
		}
		if c == '\n' {
			// LF without CR
			if d.snl != 0 {
				d.snl = snlUnexpectedLF
				d.s = sEOF
				return 0, false
			}
			d.s = sBeginLine
		}

	case sDotCR:
		if c == '\n' {
			// normal CR-LF
			d.s = sEOF
			return 0, false
		}
		// CR without LF
		d.r.UnreadByte(c) // put thing we just read back
		if d.snl != 0 {
			d.snl = snlUnexpectedCR
			d.s = sEOF
			return 0, false
		}
		c = '\r' // process previous CR instead
		d.s = sNonBegin

	case sCR:
		if c == '\n' {
			// normal CR-LF
			d.s = sBeginLine
			break
		}
		// CR without LF
		d.r.UnreadByte(c) // put thing we just read back
		if d.snl != 0 {
			d.snl = snlUnexpectedCR
			d.s = sEOF
			return 0, false
		}
		c = '\r' // process CR instead
		d.s = sNonBegin
	}
	return c, true
}

func (d *DotReader) Read(b []byte) (n int, e error) {
	l := len(b)
	r := d.r
	var c byte
	var v bool
	for n < l && d.s != sEOF {
		c, e = r.ReadByte()
		if e != nil {
			if e == io.EOF {
				e = io.ErrUnexpectedEOF
			}
			return
		}
		if c, v = d.process(c); v {
			b[n] = c
			n++
		}
	}
	if d.s == sEOF {
		// if we reached sEOF, e cannot be already set
		e = io.EOF
		// did we encounter newline format violation?
		if d.snl >= 2 {
			if d.snl == snlUnexpectedLF {
				e = errUnexpectedLF
			} else {
				e = errUnexpectedCR
			}
		}
	}
	return
}

func (d *DotReader) ReadByte() (c byte, e error) {
	r := d.r
	var v bool
	for d.s != sEOF {
		c, e = r.ReadByte()
		if e != nil {
			if e == io.EOF {
				e = io.ErrUnexpectedEOF
			}
			return
		}
		if c, v = d.process(c); v {
			break
		}
	}
	if d.s == sEOF {
		// if we reached sEOF, e cannot be already set
		e = io.EOF
		// did we encounter newline format violation?
		if d.snl >= 2 {
			if d.snl == snlUnexpectedLF {
				e = errUnexpectedLF
			} else {
				e = errUnexpectedCR
			}
		}
	}
	return
}

func (d *DotReader) Discard(s int) (n int, e error) {
	r := d.r
	var c byte
	var v bool
	for (s < 0 || n < s) && d.s != sEOF {
		c, e = r.ReadByte()
		if e != nil {
			if e == io.EOF {
				e = io.ErrUnexpectedEOF
			}
			return
		}
		if c, v = d.process(c); v {
			n++
		}
	}
	if d.s == sEOF {
		// if we reached sEOF, e cannot be already set
		e = io.EOF
		// did we encounter newline format violation?
		if d.snl >= 2 {
			if d.snl == snlUnexpectedLF {
				e = errUnexpectedLF
			} else {
				e = errUnexpectedCR
			}
		}
	}
	return
}
