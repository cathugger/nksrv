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
		encoded:  []byte("\n--X\naaa\n--XY\nbbb\n--X--"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa\n--XY\nbbb")},
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
		encoded:  []byte("\nXXX\n--X\n\n--X\n--X \n--X\nddd\n--X--  \nXXX"),
		boundary: "X",
		parts:    [][]byte{[]byte(""), []byte(""), []byte(""), []byte("ddd")},
	},
	{
		encoded:  []byte("\nXXX\n--X\n\n--XY\n--X \n--X\nddd\n--X--  \nXXX"),
		boundary: "X",
		parts:    [][]byte{[]byte("\n--XY"), []byte(""), []byte("ddd")},
	},
	{
		encoded:  []byte("\nXXX\n--X\naaa\n--X- \nbbb\n--X  \nccc\n--X--  \nXXX"),
		boundary: "X",
		parts:    [][]byte{[]byte("aaa\n--X- \nbbb"), []byte("ccc")},
	},
}

func mkpreader(data []byte, boundary string, annoy bool) *PartReader {
	var r io.Reader = bytes.NewReader(data)
	if annoy {
		r = annoyingReader{r}
	}
	return NewPartReader(r, boundary)
}

func doreading(r io.Reader, annoy bool) (b []byte, e error) {
	buf := new(bytes.Buffer)
	if annoy {
		r = annoyingReader{r}
	}
	_, e = buf.ReadFrom(r)
	b = buf.Bytes()
	return
}

func doTestNormal(t *testing.T, annoysupplier, annoyconsumer bool) {
	var e error
	for i := range normalmultipart {
		t.Logf("%d", i)
		pr := mkpreader(normalmultipart[i].encoded, normalmultipart[i].boundary, annoysupplier)
		for x := range normalmultipart[i].parts {
			e = pr.NextPart()
			if e != nil {
				t.Errorf("pr.NextPart(): expected nil error got %v", e)
			}
			var b []byte
			b, e = doreading(pr, annoyconsumer)
			if e != nil {
				t.Errorf("b.ReadFrom(): expected nil error got %v", e)
			}
			if !bytes.Equal(normalmultipart[i].parts[x], b) {
				t.Errorf("result not equal. expected %q got %q", normalmultipart[i].parts[x], b)
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

func testUnterminated(t *testing.T, asup, acon bool) {
	var e error
	pr := mkpreader([]byte("--X\naaa\n--X"), "X", asup)
	e = pr.NextPart()
	if e != nil {
		t.Errorf("pr.NextPart(): expected nil error got %v", e)
	}
	var b []byte
	b, e = doreading(pr, acon)
	if e != io.ErrUnexpectedEOF {
		t.Errorf("b.ReadFrom(): expected ErrUnexpectedEOF error got %v", e)
	}
	if !bytes.Equal([]byte("aaa"), b) {
		t.Errorf("result not equal. expected %q got %q", "aaa", b)
	}
	e = pr.NextPart()
	if e != io.ErrUnexpectedEOF {
		t.Errorf("b.ReadFrom(): expected ErrUnexpectedEOF error got %v", e)
	}
}

func TestUnterminated(t *testing.T) {
	testUnterminated(t, false, false)
	testUnterminated(t, false, true)
	testUnterminated(t, true, false)
	testUnterminated(t, true, true)
}
