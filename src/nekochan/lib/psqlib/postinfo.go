package psqlib

import (
	"encoding/json"
	"time"

	"nekochan/lib/mail"
)

type postInfo struct {
	ID        string // message identifier, hash of MessageID
	MessageID string // globally unique message identifier
	Date      time.Time

	MI messageInfo
	FI []fileInfo
}

type messageInfo struct {
	Title   string
	Author  string
	Trip    string
	Sage    bool
	Message string
}

type fileInfo struct {
	Type     string
	Size     int64
	ID       string // storename
	Thumb    string // thumbnail
	Original string // original file name
}

//// layout should be:

type partInfo struct {
	// which part it refers to? 0 means main message, 1 means first attachment
	Index int `json:"i,omitempty"`
	// it's common enough to put it there. will be empty for non-mp msgs
	ContentType string `json:"t,omitempty"`
	// whether encoding was binary. helps decide txt vs base64 use
	Binary bool `json:"b,omitempty"`
	// headers other than Content-Type and Content-Transfer-Encoding
	Headers mail.Headers `json:"h,omitempty"`
}

func (pi *partInfo) onlyIndex() bool {
	return !pi.Binary && pi.ContentType == "" && len(pi.Headers) == 0
}

type layoutInfo struct {
	data interface{}
}

func (i layoutInfo) MarshalJSON() (b []byte, err error) {
	if pi, ok := i.data.(*partInfo); ok {
		if pi.onlyIndex() {
			return json.Marshal(&pi.Index)
		} else {
			return json.Marshal(&pi)
		}
	}
	if ll, ok := i.data.([]layoutInfo); ok {
		return json.Marshal(ll)
	}
	panic("unrecognised type")
}

func (i *layoutInfo) UnmarshalJSON(b []byte) (err error) {
	var pi partInfo
	err = json.Unmarshal(b, &pi.Index)
	if err == nil {
		i.data = &pi
		return
	}
	err = json.Unmarshal(b, &pi)
	if err == nil {
		i.data = &pi
		return
	}
	var ll []layoutInfo
	err = json.Unmarshal(b, &ll)
	if err == nil {
		i.data = ll
		return
	}
	// error
	return
}
