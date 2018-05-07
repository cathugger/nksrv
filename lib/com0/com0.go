package com0

import (
	"encoding/json"
	"errors"
	"fmt"
)

// kinda like string except byte array for convenience inside

type StringByteBuf []byte

var _ json.Marshaler = (*StringByteBuf)(nil)
var _ json.Unmarshaler = (*StringByteBuf)(nil)

func (m StringByteBuf) MarshalJSON() ([]byte, error) {
	s := unsafeBytesToStr(m)
	return json.Marshal(s)
}

func (m *StringByteBuf) UnmarshalJSON(b []byte) error {
	var s string
	e := json.Unmarshal(b, &s)
	if e != nil {
		return e
	}
	*m = unsafeStrToBytes(s)
	return nil
}

// array of StringByteBuf. can be folded into one string when serializing

type ArrayOfStringByteBuf []StringByteBuf

func (m ArrayOfStringByteBuf) MarshalJSON() ([]byte, error) {
	if m == nil || len(m) == 0 {
		return nil, errors.New("must have atleast one value")
	}
	if len(m) == 1 {
		return json.Marshal(m[0])
	} else {
		return json.Marshal(([]StringByteBuf)(m))
	}
}

func (m *ArrayOfStringByteBuf) UnmarshalJSON(b []byte) error {
	e := json.Unmarshal(b, (*[]StringByteBuf)(m))
	if e == nil {
		return nil
	}
	*m = ArrayOfStringByteBuf{StringByteBuf{}}
	return json.Unmarshal(b, &(*m)[0])
}

// full message root. does not include external attachments.
type MsgRoot struct {
	Head map[string]ArrayOfStringByteBuf `json:"head"`
	Body BodyValue                       `json:"body"`
}

// only head
type MsgHead struct {
	Head map[string]ArrayOfStringByteBuf `json:"head"`
}

// only body
type MsgBody struct {
	Body BodyValue `json:"body"`
}

// can be either PlainBody or MultipartBody

type BodyValue struct {
	Value interface{}
}

var _ json.Marshaler = (*BodyValue)(nil)
var _ json.Unmarshaler = (*BodyValue)(nil)

func (r BodyValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Value)
}

func (r *BodyValue) UnmarshalJSON(b []byte) error {
	var mpb MultipartBody
	e := json.Unmarshal(b, &mpb)
	if e == nil {
		r.Value = &mpb
		return nil
	}
	var pb PlainBody
	e = json.Unmarshal(b, &pb)
	if e == nil {
		r.Value = &pb
		return nil
	}
	return e
}

// used inside PlainBody

type BodyType int

const (
	StringBody BodyType = iota
	ExternalBody
	Base64Body
	NumBody
)

var bodyTypeToString = [NumBody]string{
	StringBody:   "str",
	ExternalBody: "ext",
	Base64Body:   "b64",
}

var _ json.Marshaler = (*BodyType)(nil)
var _ json.Unmarshaler = (*BodyType)(nil)

func (t BodyType) MarshalJSON() ([]byte, error) {
	if t >= 0 && int(t) < len(bodyTypeToString) {
		return json.Marshal(bodyTypeToString[t])
	}
	return nil, fmt.Errorf("body type %d cannot be represented as string", int(t))
}

func (t *BodyType) UnmarshalJSON(b []byte) error {
	var s string
	e := json.Unmarshal(b, &s)
	if e != nil {
		return e
	}
	switch s {
	case "str":
		*t = StringBody
	case "ext":
		*t = ExternalBody
	case "b64":
		*t = Base64Body
	default:
		return fmt.Errorf("string %q cannot be converted to body type", s)
	}
	return nil
}

// non-multipart body

type InnerPlainBody struct {
	Type  BodyType      `json:"type"`
	Value StringByteBuf `json:"val"`
}

type PlainBody struct {
	InnerPlainBody
}

func (t *PlainBody) MarshalJSON() ([]byte, error) {
	if t.Type == StringBody {
		return json.Marshal(t.Value)
	} else {
		return json.Marshal(&t.InnerPlainBody)
	}
}

func (t *PlainBody) UnmarshalJSON(b []byte) error {
	e := json.Unmarshal(b, &t.Value)
	if e == nil {
		t.Type = StringBody
		return nil
	}
	return json.Unmarshal(b, &t.InnerPlainBody)
}

// multipart body

type MultipartBody []Part

// part used in multipart body

/*
 * XXX: should I do [["key", "value"],["key", "value"]] or ["key", "value", "key", "value"]?
 * I'll pick first one for now as it's kinda cleaner to implement
 */

type Part struct {
	Head []PartHeader `json:"head,omitempty"`
	Body BodyValue    `json:"body"` // omitempty doesn't quite work there
}

// individual header of part.
// part head don't use map as we need fully deterministic order

type PartHeader struct {
	Key string
	Val StringByteBuf
}

func (t *PartHeader) MarshalJSON() ([]byte, error) {
	v := [2]interface{}{&t.Key, &t.Val}
	return json.Marshal(v)
}

func (t *PartHeader) UnmarshalJSON(b []byte) error {
	v := [2]string{}
	e := json.Unmarshal(b, &v)
	if e != nil {
		return e
	}
	t.Key = v[0]
	t.Val = unsafeStrToBytes(v[1])
	return nil
}
