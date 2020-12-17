package com0

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

type jsonPair struct {
	msg MsgRoot
	str []byte
}

var jsonCases = []jsonPair{
	{
		MsgRoot{
			Head: make(map[string]ArrayOfStringByteBuf),
			Body: BodyValue{&PlainBody{InnerPlainBody{Type: StringBody, Value: []byte("xdd")}}},
		},
		[]byte(`{"head":{},"body":"xdd"}`),
	},
	{
		MsgRoot{
			Head: make(map[string]ArrayOfStringByteBuf),
			Body: BodyValue{&PlainBody{InnerPlainBody{Type: ExternalBody, Value: []byte("xdd")}}},
		},
		[]byte(`{"head":{},"body":{"type":"ext","val":"xdd"}}`),
	},
	{
		MsgRoot{
			Head: make(map[string]ArrayOfStringByteBuf),
			Body: BodyValue{&MultipartBody{}},
		},
		[]byte(`{"head":{},"body":[]}`),
	},
	{
		MsgRoot{
			Head: make(map[string]ArrayOfStringByteBuf),
			Body: BodyValue{&MultipartBody{
				{
					Head: nil,
					Body: BodyValue{&PlainBody{InnerPlainBody{Type: StringBody, Value: []byte("xdd")}}},
				},
			}},
		},
		[]byte(`{"head":{},"body":[{"body":"xdd"}]}`),
	},
	{
		MsgRoot{
			Head: make(map[string]ArrayOfStringByteBuf),
			Body: BodyValue{&MultipartBody{
				{
					Head: []PartHeader{
						{
							Key: "testkey1",
							Val: []byte("testval1"),
						},
					},
					Body: BodyValue{&PlainBody{InnerPlainBody{Type: StringBody, Value: []byte("xdd")}}},
				},
			}},
		},
		[]byte(`{"head":{},"body":[{"head":[["testkey1","testval1"]],"body":"xdd"}]}`),
	},
	{
		MsgRoot{
			Head: map[string]ArrayOfStringByteBuf{
				"a": {[]byte("aval")},
			},
			Body: BodyValue{&MultipartBody{
				{
					Head: []PartHeader{
						{
							Key: "testkey1",
							Val: []byte("testval1"),
						},
						{
							Key: "testkey1",
							Val: []byte("testval2"),
						},
						{
							Key: "testkey2",
							Val: []byte("testval3"),
						},
					},
					Body: BodyValue{&PlainBody{InnerPlainBody{Type: StringBody, Value: []byte("xdd")}}},
				},
			}},
		},
		[]byte(`{"head":{"a":"aval"},"body":[{"head":[["testkey1","testval1"],["testkey1","testval2"],["testkey2","testval3"]],"body":"xdd"}]}`),
	},
	{
		MsgRoot{
			Head: map[string]ArrayOfStringByteBuf{
				"a": {[]byte("aval")},
				"b": {[]byte("bval1"), []byte("bval2"), []byte("bval3")},
				"c": {[]byte("cval")},
				"d": {[]byte("dval")},
			},
			Body: BodyValue{&MultipartBody{
				{
					Head: nil,
					Body: BodyValue{&PlainBody{InnerPlainBody{Type: StringBody, Value: []byte("xdd")}}},
				},
			}},
		},
		[]byte(`{"head":{"a":"aval","b":["bval1","bval2","bval3"],"c":"cval","d":"dval"},"body":[{"body":"xdd"}]}`),
	},
}

func TestJSONMarshal(t *testing.T) {
	for i := range jsonCases {
		t.Logf("testcase %d", i)
		b, e := json.Marshal(jsonCases[i].msg)
		if e != nil {
			t.Fatal("json.Marshal failed:", e)
		}
		t.Logf("marshal output: %s", b)
		if !bytes.Equal(b, jsonCases[i].str) {
			t.Fatalf("provided %q is not same as resulting %q", jsonCases[i].str, b)
		}
	}
}

func TestJSONUnmarshal(t *testing.T) {
	for i := range jsonCases {
		t.Logf("testcase %d", i)
		var v MsgRoot
		e := json.Unmarshal(jsonCases[i].str, &v)
		if e != nil {
			t.Fatal("json.Unmarshal failed:", e)
		}
		t.Logf("unmarshal output: %s", spew.Sdump(v))
		if !reflect.DeepEqual(v, jsonCases[i].msg) {
			t.Fatalf("provided struct is not same as resulting struct")
		}
	}
}
