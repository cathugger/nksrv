package mail

import (
	"math/rand"
	"mime"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func isBadChar(r rune) bool {
	return r >= 0x7F || (r < 0x20 && r != '\t')
}

func doTest(t *testing.T, i int, testmap map[string]string) {
	x := FormatMediaTypeX("x", testmap)
	if x == "" {
		t.Errorf("FormatMediaTypeX failed on %d", i)
		return
	}
	if strings.IndexFunc(x, isBadChar) >= 0 {
		t.Errorf("bad char found on %d: %q", i, x)
		return
	}
	typ, m, e := mime.ParseMediaType(x)
	if e != nil {
		t.Errorf("ParseMediaType failed on %d %q: %v", i, x, e)
		return
	}
	if typ != "x" {
		t.Errorf("ParseMediaType wrong type on %d", i)
		return
	}
	if !reflect.DeepEqual(testmap, m) {
		t.Errorf(
			"ParseMediaType not deep equal on %d: wanted %#v got %#v",
			i, testmap, m)
		return
	}
}

func TestFormatMediaType(t *testing.T) {
	tests := []map[string]string{
		{"x": "y"},
		{"x": " y y y y y y "},
		{"x": "Брэд"},
		{"filename": "数据统计.png"},
		{"filename1": "数据统计.png", "filename2": "数据统计.jpg"},
		{"filename": "test.png"},
	}
	for i := range tests {
		doTest(t, i, tests[i])
	}
}

func genRandString(r *rand.Rand) string {
	length := int(r.Int31n(128))
	runes := make([]rune, length)
	for i := 0; i < length; {
		run := r.Int31n(0x10FFFF + 1)
		if !utf8.ValidRune(run) {
			continue
		}
		runes[i] = run
		i++
	}
	return string(runes)
}

func TestFormatMediaTypeRand(t *testing.T) {
	r := rand.New(rand.NewSource(666))
	const numTest = 2048

	for i := 0; i < numTest; i++ {
		ra, rb := genRandString(r), genRandString(r)
		rc, rd := genRandString(r), genRandString(r)

		test := map[string]string{
			"a": ra, "b": rb,
			"c": rc, "d": rd,
		}
		doTest(t, i, test)
	}
}
