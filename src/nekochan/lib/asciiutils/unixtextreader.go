package asciiutils

import "io"

type UnixTextReader struct {
	u io.Reader
	s byte
}

func NewUnixTextReader(r io.Reader) *UnixTextReader {
	return &UnixTextReader{u: r}
}

// strips all \r and ensures text ends with \n
func (t *UnixTextReader) Read(b []byte) (n int, e error) {
	const (
		sNL byte = iota
		sText
		sEOF
	)
	if len(b) == 0 {
		return
	}
	if t.s == sEOF {
		b[0] = '\n'
		n = 1
		e = io.EOF
		t.s = sNL
		return
	}
	var i, x int
	x, e = t.u.Read(b)
	for i = 0; i < x && b[i] != '\r'; {
		if b[i] != '\n' {
			t.s = sText
		} else {
			t.s = sNL
		}
		i++
		n++
	}
	for ; i < x; i++ {
		c := b[i]
		if c == '\r' {
			continue
		}

		b[n] = c
		n++
		if c != '\n' {
			t.s = sText
		} else {
			t.s = sNL
		}
	}
	if e == io.EOF && t.s != sNL {
		// needs additional newline at the end
		if len(b) > n {
			// can insert
			b[n] = '\n'
			n++
			t.s = sNL
		} else {
			// can't end stuff there
			e = nil
			t.s = sEOF
		}
	}
	return
}
