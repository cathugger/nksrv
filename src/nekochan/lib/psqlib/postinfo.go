package psqlib

import (
	"encoding/json"
	"fmt"
	"time"

	"nekochan/lib/mail"
)

type postInfo struct {
	ID        string // message identifier, hash of MessageID
	MessageID string // globally unique message identifier
	Date      time.Time

	MI messageInfo
	FI []fileInfo

	H mail.Headers
	L partInfo
}

type messageInfo struct {
	Title   string
	Author  string
	Trip    string
	Sage    bool
	Message string
}

type FTypeT int

const (
	FTypeFile FTypeT = iota
	FTypeMsg
	FTypeText
	FTypeImage
)

var FTypeS = map[FTypeT]string{
	FTypeFile:  "file",
	FTypeMsg:   "msg",
	FTypeText:  "text",
	FTypeImage: "image",
}

type fileInfo struct {
	Type        FTypeT
	ContentType string // MIME type (without parameters)
	Size        int64
	ID          string // storename
	Thumb       string // thumbnail
	Original    string // original file name
}

//// layout should be:

/*

0

{
	"h": {"h1": ["v1","v2"]},
	"b": 0
}

{
	"h": {"h1": ["v1","v2"]},
	"b": [
		{
			"h": {"h1": ["v1","v2"]},
			"b": 0
		},
		{
			"h": {"h1": ["v1","v2"]},
			"b": 1
		},
		2
	]
}

*/

type postObjectIndex = uint32

type bodyObject struct {
	// one of postObjectIndex, []partInfo, nil
	Data interface{}
}

func (i *bodyObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Data)
}

func (i *bodyObject) UnmarshalJSON(b []byte) (err error) {
	var poi postObjectIndex
	err = json.Unmarshal(b, &poi)
	if err == nil {
		i.Data = poi
		return
	}
	var parts []partInfo
	err = json.Unmarshal(b, &parts)
	if err == nil {
		i.Data = parts
		return
	}
	var null interface{}
	err = json.Unmarshal(b, &null)
	if err == nil {
		if null == nil {
			i.Data = nil
			return
		} else {
			return fmt.Errorf("bodyObject: unexpected unmarshal: %#v", null)
		}
	}
	// error
	return
}

type partInfoInner struct {
	ContentType string       `json:"t,omitempty"`
	Binary      bool         `json:"x,omitempty"`
	Headers     mail.Headers `json:"h,omitempty"`
	Body        bodyObject   `json:"b"`
}

func (i *partInfoInner) onlyBody() bool {
	return i.ContentType == "" && !i.Binary && len(i.Headers) == 0
}

type partInfo struct {
	partInfoInner
}

func (i *partInfo) MarshalJSON() ([]byte, error) {
	if i.onlyBody() {
		return json.Marshal(i.Body) // array or integer
	}
	return json.Marshal(i.partInfoInner)
}

func (i *partInfo) UnmarshalJSON(b []byte) (err error) {
	err = json.Unmarshal(b, &i.partInfoInner)
	if err == nil {
		return
	}
	err = json.Unmarshal(b, &i.Body)
	if err == nil {
		return
	}
	// error
	return
}
