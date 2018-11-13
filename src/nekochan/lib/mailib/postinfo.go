package mailib

import (
	"encoding/json"
	"fmt"
	"time"

	"nekochan/lib/mail"
)

type PostInfo struct {
	ID        string // message identifier, hash of MessageID
	MessageID string // globally unique message identifier
	Date      time.Time

	MI MessageInfo
	FI []FileInfo

	H mail.Headers
	L PartInfo
}

type MessageInfo struct {
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

type FileInfo struct {
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

type PostObjectIndex = uint32

type BodyObject struct {
	// one of PostObjectIndex, []PartInfo, nil
	Data interface{}
}

func (i *BodyObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Data)
}

func (i *BodyObject) UnmarshalJSON(b []byte) (err error) {
	var poi PostObjectIndex
	err = json.Unmarshal(b, &poi)
	if err == nil {
		i.Data = poi
		return
	}
	var parts []PartInfo
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
			return fmt.Errorf("BodyObject: unexpected unmarshal: %#v", null)
		}
	}
	// error
	return
}

type PartInfoInner struct {
	ContentType string       `json:"t,omitempty"`
	Binary      bool         `json:"x,omitempty"`
	Headers     mail.Headers `json:"h,omitempty"`
	Body        BodyObject   `json:"b"`
}

func (i *PartInfoInner) onlyBody() bool {
	return i.ContentType == "" && !i.Binary && len(i.Headers) == 0
}

type PartInfo struct {
	PartInfoInner
}

func (i *PartInfo) MarshalJSON() ([]byte, error) {
	if i.onlyBody() {
		return json.Marshal(i.Body) // array or integer
	}
	return json.Marshal(i.PartInfoInner)
}

func (i *PartInfo) UnmarshalJSON(b []byte) (err error) {
	err = json.Unmarshal(b, &i.PartInfoInner)
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