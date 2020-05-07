package webib0

import "nksrv/lib/mail"

// web IB API representation v0

// references to other places than current thread will need to be hacked in
// references to curren thread' objects are easy because we can drop
// them in on rendering stage but for foreign ones we can't know
// instead of storing all references, we should save some space and
// store only when they refer to foreign stuff
// so, basically, aid renderer only when it would be at loss
// we should hack them in to database and calculate only at post time
// to minimize complexity at query time

// board list member
type IBBoardListBoard struct {
	BNum        uint64   `json:"bn"`             // internal num
	Name        string   `json:"name"`           // short name
	Description string   `json:"desc"`           // description
	Tags        []string `json:"tags,omitempty"` // tags
	NumThreads  int64    `json:"num_threads"`
	NumPosts    int64    `json:"num_posts"`
}

// board list page
type IBBoardList struct {
	Boards []IBBoardListBoard `json:"boards"` // boards
}

type IBThumbAttributes struct {
	Width  uint32 `json:"w,omitempty"` // width
	Height uint32 `json:"h,omitempty"` // height
}

// thumbnail info
// consistent across pages
type IBThumbInfo struct {
	ID  string `json:"id,omitempty"`  // identifier. filename
	Alt string `json:"alt,omitempty"` // alternative. for stuff like spoilers
	IBThumbAttributes
}

// file info
// consistent across pages
type IBFileInfo struct {
	ID       string                 `json:"id"`             // identifier. filename
	Type     string                 `json:"type"`           // short type of file, kind
	Thumb    IBThumbInfo            `json:"thumb"`          // thumbnail
	Original string                 `json:"orig"`           // original filename
	Size     int64                  `json:"size"`           // all files have size
	Options  map[string]interface{} `json:"opts,omitempty"` // metadata which depends on file type
}

type IBReference struct {
	Board  string `json:"b,omitempty"` // board which contains post which is refered to. if empty, "this board"
	Thread string `json:"t,omitempty"` // thread which contains post which is refered to. if empty, "this thread"
	Post   string `json:"p,omitempty"` // full external post id. may be empty when referencing to board or thread
}

type IBMessageReference struct {
	Start uint `json:"s"` // points to reference start position in Message
	End   uint `json:"e"` // points after reference end in Message
	IBReference
}

type IBBackReference struct {
	IBReference
}

type IBMessage []byte

// post
type IBPostInfo struct {
	Num            uint64                 `json:"n"`                 // internal post number. unique only per this server
	ID             string                 `json:"id"`                // external ID of post. untruncated, per board
	MsgID          string                 `json:"msgid"`             // Message-ID of post, if available
	Subject        string                 `json:"subject"`           // subject text
	Name           string                 `json:"name"`              // name of poster
	Trip           string                 `json:"trip,omitempty"`    // tripcode, usually not set
	Email          string                 `json:"email,omitempty"`   // email field, usually useless, used for sage too
	Sage           bool                   `json:"sage,omitempty"`    // sage
	Date           int64                  `json:"date"`              // seconds since unix epoch
	Message        IBMessage              `json:"msg"`               // message itself. formatted
	References     []IBMessageReference   `json:"refs,omitempty"`    // posts Message refers to
	Files          []IBFileInfo           `json:"files,omitempty"`   // attached files
	BackReferences []IBBackReference      `json:"brefs,omitempty"`   // post refering to this post
	Headers        mail.HeaderMap         `json:"headers,omitempty"` // headers
	Options        map[string]interface{} `json:"opts,omitempty"`    // additional stuff
}

func (pi *IBPostInfo) PubKey() string {
	if len(pi.Trip) != 0 && pi.Trip[0] != '!' {
		return pi.Trip
	}
	return ""
}

// common thread fields
type IBCommonThread struct {
	ID      string                 `json:"id"`                // thread ID
	OP      IBPostInfo             `json:"op"`                // OP
	Replies []IBPostInfo           `json:"replies,omitempty"` // replies
	Options map[string]interface{} `json:"opts,omitempty"`    // additional stuff
}

// thread in thread list page
type IBThreadListPageThread struct {
	SkippedReplies int64 `json:"skipreplies,omitempty"` // number of replies not included
	SkippedFiles   int64 `json:"skipfiles,omitempty"`   // number of attachments not included

	IBCommonThread
}

// info about board common across pages
type IBBoardInfo struct {
	BNum        uint32 `json:"bn"`   // internal board num
	Name        string `json:"name"` // short name
	Description string `json:"desc"` // description
	Info        string `json:"info"` // optional additional info string
}

type IBThreadListPage struct {
	Board       IBBoardInfo              `json:"board"`              // info about this board
	Number      uint32                   `json:"pnum"`               // this page num (starting from 0)
	Available   uint32                   `json:"pavail"`             // num of pages
	Threads     []IBThreadListPageThread `json:"threads,omitempty"`  // threads
	HasBackRefs bool                     `json:"hasbrefs,omitempty"` // whether backreferences are already calculated
}

type IBOverboardPageThread struct {
	BNum      uint32 `json:"bn"` // internal board num
	BoardName string `json:"bname"`

	IBThreadListPageThread
}

type IBOverboardPage struct {
	Number      uint32                  `json:"pnum"`   // this page num (starting from 0)
	Available   uint32                  `json:"pavail"` // num of pages
	Threads     []IBOverboardPageThread `json:"threads"`
	HasBackRefs bool                    `json:"hasbrefs,omitempty"` // whether backreferences are already calculated
}

type IBThreadStats struct {
	NumReplies int64  `json:"reply_count"` // not counting OP
	NumFiles   int64  `json:"file_count"`  // including OP
	PageNum    uint32 `json:"page_num"`    // starting from 0
}

type IBThreadPage struct {
	Board       IBBoardInfo   `json:"board"`              // info about this board
	HasBackRefs bool          `json:"hasbrefs,omitempty"` // whether backreferences are already calculated
	ThreadStats IBThreadStats `json:"stats"`

	IBCommonThread
}

type IBThreadCatalogThread struct {
	Num          uint64      `json:"n"`        // internal thread number
	ID           string      `json:"id"`       // external thread ID
	Thumb        IBThumbInfo `json:"thumb"`    // thumbnail
	TotalReplies int64       `json:"nreplies"` // number of replies
	TotalFiles   int64       `json:"nfiles"`   // number of attachments
	BumpDate     int64       `json:"bdate"`    // bump date
	Subject      string      `json:"subject"`  // subject
	Message      IBMessage   `json:"msg"`      // message
}

type IBThreadCatalog struct {
	Board   IBBoardInfo             `json:"board"`             // info about this baord
	Threads []IBThreadCatalogThread `json:"threads,omitempty"` // threads
}

type IBOverboardCatalogThread struct {
	BNum      uint32 `json:"bn"` // internal board num
	BoardName string `json:"bname"`

	IBThreadCatalogThread
}

type IBOverboardCatalog struct {
	Threads []IBOverboardCatalogThread `json:"threads,omitempty"` // threads
}
