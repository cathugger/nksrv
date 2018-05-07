package com0

import (
	"bytes"
	"encoding/json"
	spew "github.com/davecgh/go-spew/spew"
	"reflect"
	"testing"
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
