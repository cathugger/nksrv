package mail

import (
	"bytes"
	"io"
	"testing"
)

type annoyingReader struct {
	io.Reader
}

func (a annoyingReader) Read(b []byte) (int, error) {
	if len(b) != 0 {
		b = b[:1]
	}
	return a.Reader.Read(b)
}

var normalmultipart = []struct {
	encoded  []byte
	boundary string
	parts    [][]byte
}{
	{
		encoded:  []byte("--X\r\naaa\r\n--X--\r\n"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa")},
	},
	{
		encoded:  []byte("--X\r\naaa\r\n--X--"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa")},
	},
	{
		encoded:  []byte("--X\r\naaa\r\n\r\n--X--"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa\r\n")},
	},
	{
		encoded:  []byte("--X\naaa\n--X--\n"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa")},
	},
	{
		encoded:  []byte("--X\naaa\n--X--"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa")},
	},
	{
		encoded:  []byte("--X\naaa\n\n--X--"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa\n")},
	},
	{
		encoded:  []byte("beeep\nbooop\n--X\naaa\n\n--X--"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa\n")},
	},
	{
		encoded:  []byte("--XX\n--X\naaa\n--X--"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa")},
	},
	{
		encoded:  []byte("\n--XX\n--X\naaa\n--X--\nabcdabcd\nabcd"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa")},
	},
	{
		encoded:  []byte("\n--XX\n--X\naaa\n--X\nbbb\n--X\nccc\n--X--\nabcdabcd\nabcd"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa"), []byte("bbb"), []byte("ccc")},
	},
	{
		encoded:  []byte("\nXXX\n--X\naaa\n--X \nbbb\n--X  \nccc\n--X--  \nXXX"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa"), []byte("bbb"), []byte("ccc")},
	},
	{
		encoded:  []byte("\nXXX\n--X\naaa\n--X- \nbbb\n--X  \nccc\n--X--  \nXXX"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa\n--X- \nbbb"), []byte("ccc")},
	},
}

func doTestNormal(t *testing.T, annoysupplier, annoyconsumer bool) {
	var e error
	for i := range normalmultipart {
		t.Logf("%d", i)
		var r io.Reader = bytes.NewReader(normalmultipart[i].encoded)
		if annoysupplier {
			r = annoyingReader{r}
		}
		pr := NewPartReader(r, normalmultipart[i].boundary)
		for x := range normalmultipart[i].parts {
			e = pr.NextPart()
			if e != nil {
				t.Errorf("pr.NextPart(): expected nil error got %v", e)
			}
			b := new(bytes.Buffer)
			if annoyconsumer {
				_, e = b.ReadFrom(annoyingReader{pr})
			} else {
				_, e = b.ReadFrom(pr)
			}
			if e != nil {
				t.Errorf("b.ReadFrom(): expected nil error got %v", e)
			}
			if !bytes.Equal(normalmultipart[i].parts[x], b.Bytes()) {
				t.Errorf("result not equal. expected %q got %q", normalmultipart[i].parts[x], b.Bytes())
			}
		}
		e = pr.NextPart()
		if e != io.EOF {
			t.Errorf("pr.NextPart(): expected EOF error got %v", e)
		}
	}
}

func TestNormal(t *testing.T) {
	doTestNormal(t, false, false)
}

func TestAnnoyingSupplier(t *testing.T) {
	doTestNormal(t, true, false)
}

func TestAnnoyingConsumer(t *testing.T) {
	doTestNormal(t, false, true)
}

func TestAnnoyingBoth(t *testing.T) {
	doTestNormal(t, true, true)
}
