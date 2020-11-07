package mailib

import (
	"encoding/json"
	"fmt"
	"time"

	"nksrv/lib/app/base/ftypes"
	"nksrv/lib/app/base/ibattribs"
	"nksrv/lib/mail"
)

type PostInfo struct {
	ID        string        // message identifier, hash of MessageID
	MessageID TCoreMsgIDStr // globally unique message identifier
	Date      time.Time

	MI MessageInfo
	FI []FileInfo
	FC int // file count -- may be less than len(FI)

	H  mail.HeaderMap
	GA GlobalPostAttribs
	BA BoardPostAttribs
	L  PartInfo
	E  PostExtraAttribs
}

type IBThumbAttribs = ibattribs.ThumbAttribs

type MessageInfo struct {
	Title   string
	Author  string
	Message string
	Sage    bool
	Trip    string
}

type GlobalPostAttribs = ibattribs.GlobalPostAttribs

type PostExtraAttribs struct {
	// if msg txt is in attachment, 1-based index which file it is
	TextAttachment uint32 `json:"text_attach,omitempty"`
}

type BoardPostAttribs = ibattribs.BoardPostAttribs

type FileExtraAttribs struct {
	ContentType string `json:"ct,omitempty"`
}

type FileInfo struct {
	Type        ftypes.FTypeT // kind
	ContentType string        // MIME type (without parameters)
	Size        int64
	ID          string                 // storename
	ThumbField  string                 // thumbnail suffix stored in db
	Original    string                 // original file name
	FileAttrib  map[string]interface{} // file attributes
	ThumbAttrib IBThumbAttribs         // thumbnail attributes
	Extras      FileExtraAttribs
}

func (x FileInfo) Equivalent(y FileInfo) bool {
	return x.ID == y.ID && x.Original == y.Original && x.Size == y.Size
}

type TThumbInfo struct {
	FullTmpName string
	RelDestName string
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

func (i BodyObject) MarshalJSON() ([]byte, error) {
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
	ContentType string            `json:"t,omitempty"`
	Binary      bool              `json:"x,omitempty"`
	HasNull     bool              `json:"0,omitempty"` // not used if Binary
	Has8Bit     bool              `json:"8,omitempty"` // not used if Binary or HasNull
	Headers     mail.HeaderMap    `json:"h,omitempty"`
	MPParams    map[string]string `json:"m,omitempty"`
	Body        BodyObject        `json:"b"`
}

func (i *PartInfoInner) onlyBody() bool {
	return i.ContentType == "" &&
		i.Binary == false &&
		i.HasNull == false &&
		i.Has8Bit == false &&
		len(i.Headers) == 0 &&
		len(i.MPParams) == 0
}

type PartInfo struct {
	PartInfoInner
}

func (i *PartInfo) MarshalJSON() ([]byte, error) {
	if i.onlyBody() {
		return json.Marshal(i.Body) // array or integer
	}
	return json.Marshal(&i.PartInfoInner)
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
