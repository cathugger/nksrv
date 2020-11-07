// XXX move to its own package?

package bufreader

import "io"

type DotReader struct {
	r     *BufReader // underlying reader
	s     int        // parser state
	badnl bool       // whether we encountered unmatching CR/LF
}

func NewDotReader(r *BufReader) *DotReader {
	return &DotReader{r: r}
}

func (d *DotReader) InvalidNL() bool {
	return d.badnl
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
	d.badnl = false
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
			d.badnl = true
			d.s = sBeginLine
		}

	case sDot:
		if c == '\r' {
			d.s = sDotCR
			return 0, false
		}
		if c == '\n' {
			// LF after dot
			d.badnl = true
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
			d.badnl = true
			d.s = sBeginLine
		}

	case sDotCR:
		if c == '\n' {
			// normal CR-LF
			d.s = sEOF
			return 0, false
		}
		// CR without LF
		d.badnl = true
		d.r.UnreadByte(c) // put thing we just read back
		c = '\r'          // process previous CR instead
		d.s = sNonBegin

	case sCR:
		if c == '\n' {
			// normal CR-LF
			d.s = sBeginLine
			break
		}
		// CR without LF
		d.badnl = true
		d.r.UnreadByte(c) // put thing we just read back
		c = '\r'          // process CR instead
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
	}
	return
}
