package mail

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

type hr_testcase struct {
	msg   []byte
	limit int64
	hdrs  Headers
}

var hr_tests = []hr_testcase{
	{
		msg:   []byte("\n"),
		limit: 0,
		hdrs:  Headers{},
	},
	{
		msg:   []byte("\nsomething"),
		limit: 0,
		hdrs:  Headers{},
	},
	{
		msg:   []byte("A: b\n\n"),
		limit: 0,
		hdrs:  Headers{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("A:b\n\n"),
		limit: 0,
		hdrs:  Headers{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("A   :b\n\n"),
		limit: 0,
		hdrs:  Headers{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("a: b\n\n"),
		limit: 0,
		hdrs: Headers{
			"A": HeaderVals{{HeaderValInner: HeaderValInner{V: "b", O: "a"}}},
		},
	},
	{
		msg:   []byte("A: b\n c\n d\n\n"),
		limit: 0,
		hdrs:  Headers{"A": OneHeaderVal("b c d")},
	},
	{
		msg:   []byte("A: b\n\tc\n\td\n\n"),
		limit: 0,
		hdrs:  Headers{"A": OneHeaderVal("b\tc\td")},
	},
}

func init() {
	long := hr_testcase{}

	b := new(bytes.Buffer)
	for i := 0; i < 16000; i++ {
		fmt.Fprintf(b, "a-%d-b", i)
	}
	long.msg = append([]byte("A: "), b.Bytes()...)
	long.msg = append(long.msg, []byte("\n\n")...)
	long.hdrs = Headers{"A": OneHeaderVal(string(b.Bytes()))}

	hr_tests = append(hr_tests, long)
}

func TestValid(t *testing.T) {
	for i := range hr_tests {
		br := bytes.NewReader(hr_tests[i].msg)
		mh, e := ReadHeaders(br, hr_tests[i].limit)
		if e != nil {
			t.Errorf("ReadHeaders err: %v", e)
		}
		defer mh.Close()
		if !reflect.DeepEqual(mh.H, hr_tests[i].hdrs) {
			t.Errorf("not equal: got %#v expected %#v", mh.H, hr_tests[i].hdrs)
		}
	}
}
