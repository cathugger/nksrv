package asciiutils

import (
	"bytes"
	"io"
	"testing"
)

var readcases = []struct {
	a, b string
}{
	{"aaa\n", "aaa\n"},
	{"bbb", "bbb\n"},
	{"ccc\r\n", "ccc\n"},
	{"ddd\r", "ddd\n"},
	{"ddd\ree", "dddee\n"},
	{"ddd\ree\n", "dddee\n"},
	{"\r\r\rddd", "ddd\n"},
}

type annoyingReader struct {
	io.Reader
	x bool
}

func (a *annoyingReader) Read(b []byte) (int, error) {
	if len(b) != 0 {
		if a.x {
			b = b[:1]
		} else {
			b = b[:0]
		}
		a.x = !a.x
	}
	return a.Reader.Read(b)
}

func testTextReader(t *testing.T, asup, acon bool) {
	for _, c := range readcases {
		var r io.Reader = bytes.NewReader([]byte(c.a))
		if asup {
			r = &annoyingReader{Reader: r}
		}
		var tr io.Reader = NewUnixTextReader(r)
		if acon {
			tr = &annoyingReader{Reader: tr}
		}
		b := &bytes.Buffer{}
		b.ReadFrom(tr)
		br := b.Bytes()
		if !bytes.Equal([]byte(c.b), br) {
			t.Errorf("not equal, expected %q got %q", c.b, br)
		}
	}
}

func TestTextReader(t *testing.T) {
	testTextReader(t, false, false)
	testTextReader(t, false, true)
	testTextReader(t, true, false)
	testTextReader(t, true, true)
}
