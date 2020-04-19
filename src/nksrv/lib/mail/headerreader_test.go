package mail

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

type hr_testcase struct {
	msg   []byte
	limit int
	hdrs  HeaderMap
}

var hr_tests = []hr_testcase{
	{
		msg:   []byte("\n"),
		limit: 0,
		hdrs:  HeaderMap{},
	},
	{
		msg:   []byte("\nsomething"),
		limit: 0,
		hdrs:  HeaderMap{},
	},
	{
		msg:   []byte("A: b\n\n"),
		limit: 0,
		hdrs:  HeaderMap{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("A:b\n\n"),
		limit: 0,
		hdrs:  HeaderMap{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("A   :b\n\n"),
		limit: 0,
		hdrs:  HeaderMap{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("a: b\n\n"),
		limit: 0,
		hdrs: HeaderMap{
			"A": HeaderMapVals{{
				HeaderMapValInner: HeaderMapValInner{V: "b", O: "a"}}},
		},
	},
	{
		msg:   []byte("A:    b\n\n"),
		limit: 0,
		hdrs:  HeaderMap{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("A: b    \n\n"),
		limit: 0,
		hdrs:  HeaderMap{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("A:     b    \n\n"),
		limit: 0,
		hdrs:  HeaderMap{"A": OneHeaderVal("b")},
	},
	{
		msg:   []byte("A: b\n c\n d\n\n"),
		limit: 0,
		hdrs: HeaderMap{"A": HeaderMapVals{{HeaderMapValInner{
			V: "b c d",
			S: HeaderValSplitList{1, 2},
		}}}},
	},
	{
		msg:   []byte("A: b\n c\n d\n e\nB: b\n c\n d\n e\n\n"),
		limit: 0,
		hdrs: HeaderMap{
			"A": HeaderMapVals{{HeaderMapValInner{
				V: "b c d e",
				S: HeaderValSplitList{1, 2, 2},
			}}},
			"B": HeaderMapVals{{HeaderMapValInner{
				V: "b c d e",
				S: HeaderValSplitList{1, 2, 2},
			}}},
		},
	},
	{
		msg:   []byte("A: b\n\tc\n\td\n\n"),
		limit: 0,
		hdrs: HeaderMap{
			"A": HeaderMapVals{{HeaderMapValInner{
				V: "b\tc\td",
				S: HeaderValSplitList{1, 2},
			}}},
		},
	},
}

func init() {

	br := new(bytes.Buffer)
	bt := new(bytes.Buffer)

	for i := 0; i < 13000; i++ {
		tc := hr_testcase{}
		br.Reset()
		for j := 0; j < i; j++ {
			c := rune(0x23 + (i-j)%(0x26-0x23+1))
			fmt.Fprintf(br, "%c", c)
		}
		tc.msg = append([]byte(fmt.Sprintf("A%05d: ", i)), br.Bytes()...)
		tc.msg = append(tc.msg, []byte("\n\n")...)
		tc.hdrs = HeaderMap{fmt.Sprintf("A%05d", i): OneHeaderVal(string(br.Bytes()))}
		hr_tests = append(hr_tests, tc)
	}

	for i := 1; i < 13000; i++ {
		tc := hr_testcase{}
		br.Reset()
		bt.Reset()
		for j := 0; j < 13000; j++ {
			if j == i {
				fmt.Fprintf(br, "\n ")
				fmt.Fprintf(bt, " ")
			}
			c := rune(0x23 + (13000-j+i)%(0x26-0x23+1))
			fmt.Fprintf(br, "%c", c)
			fmt.Fprintf(bt, "%c", c)
		}

		id1 := fmt.Sprintf("B1-%05d", i)
		id2 := fmt.Sprintf("B2-%05d", i)
		id3 := fmt.Sprintf("B3-%05d", i)

		tc.msg = append(tc.msg, []byte(id1)...)
		tc.msg = append(tc.msg, []byte(": ")...)
		tc.msg = append(tc.msg, br.Bytes()...)
		tc.msg = append(tc.msg, byte('\n'))

		tc.msg = append(tc.msg, []byte(id2)...)
		tc.msg = append(tc.msg, []byte(": ")...)
		tc.msg = append(tc.msg, []byte("A")...)
		tc.msg = append(tc.msg, br.Bytes()...)
		tc.msg = append(tc.msg, byte('\n'))

		tc.msg = append(tc.msg, []byte(id3)...)
		tc.msg = append(tc.msg, []byte(": ")...)
		tc.msg = append(tc.msg, []byte("AA")...)
		tc.msg = append(tc.msg, br.Bytes()...)
		tc.msg = append(tc.msg, byte('\n'))

		tc.msg = append(tc.msg, byte('\n'))

		tc.hdrs = HeaderMap{
			id1: HeaderMapVals{{HeaderMapValInner{
				V: string(bt.Bytes()),
				S: HeaderValSplitList{HeaderValSplit(i)},
			}}},
			id2: HeaderMapVals{{HeaderMapValInner{
				V: "A" + string(bt.Bytes()),
				S: HeaderValSplitList{HeaderValSplit(i + 1)},
			}}},
			id3: HeaderMapVals{{HeaderMapValInner{
				V: "AA" + string(bt.Bytes()),
				S: HeaderValSplitList{HeaderValSplit(i + 2)},
			}}},
		}
		hr_tests = append(hr_tests, tc)
	}

}

func TestValid(t *testing.T) {
	const which = -1
	for i := range hr_tests {
		if which >= 0 && i != which {
			continue
		}
		br := bytes.NewReader(hr_tests[i].msg)
		mh, e := ReadHeaders(br, hr_tests[i].limit)
		if e != nil {
			t.Fatalf("%d ReadHeaders err: %v", i, e)
		}
		defer mh.Close()
		if !reflect.DeepEqual(mh.H, hr_tests[i].hdrs) {
			t.Logf("%d not equal!", i)
			t.Logf("got %#v", mh.H)
			t.Logf("expected %#v", hr_tests[i].hdrs)
			t.Logf("input %q", hr_tests[i].msg)
			t.FailNow()
		}
	}
}
