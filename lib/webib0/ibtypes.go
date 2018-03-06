package webib0

// web IB API representation v0

// references to other places than current thread will need to be hacked in
// references to curren thread' objects are easy because we can drop
// them in on rendering stage but for foreign ones we can't know
// instead of storing all references, we should save some space and
// store only when they refer to foreign stuff
// so, basically, aid renderer only when it would be at loss
// we should hack them in to database and calculate only at post time
// to minimize complexity at query time

// information about this node
type IBNodeInfo struct {
	Name  string `json:"name"`
	Root  string `json:"root"`
	FRoot string `json:"froot"`
}

// board list member
type IBBoardListBoard struct {
	Name        string   `json:"name"` // short name
	Description string   `json:"desc"` // description
	Tags        []string `json:"tags"` // tags
}

// board list page
type IBBoardList struct {
	Node   IBNodeInfo         `json:"node"`   // info about this node
	Boards []IBBoardListBoard `json:"boards"` // boards
}

// thumbnail info
// consistent across pages
type IBThumbInfo struct {
	ID     string `json:"id,omitempty"`     // identifier. filename
	Alt    string `json:"alt,omitempty"`    // alternative. for stuff like spoilers
	Width  uint32 `json:"width,omitempty"`  // width
	Height uint32 `json:"height,omitempty"` // height
}

// file info
// consistent across pages
type IBFileInfo struct {
	ID       string                 `json:"id"`                // identifier. filename
	Type     string                 `json:"type"`              // short type of file
	Thumb    IBThumbInfo            `json:"thumb"`             // thumbnail
	Original string                 `json:"original"`          // original filename
	Size     int64                  `json:"size"`              // all files have size
	Options  map[string]interface{} `json:"options,omitempty"` // metadata which depends on file type
}

type IBReference struct {
	Start  uint   `json:"start"`            // points to reference start position in Message
	End    uint   `json:"end"`              // points after reference end in Message
	Board  string `json:"board,omitempty"`  // board which contains post which is refered to. if empty, "this board"
	Thread string `json:"thread,omitempty"` // thread which contains post which is refered to. if empty, "this thread"
	Post   string `json:"post"`             // full post number
}

// post
type IBPostInfo struct {
	ID             string                 `json:"id"`                       // ID of post. long, global one
	Subject        string                 `json:"subject"`                  // subject text
	Name           string                 `json:"name"`                     // name of poster
	Trip           string                 `json:"trip,omitempty"`           // tripcode, usually not set
	Email          string                 `json:"email,omitempty"`          // email field, usually useless, used for sage too
	Date           int64                  `json:"date"`                     // seconds since unix epoch
	Message        []byte                 `json:"message"`                  // message itself. formatted
	References     []IBReference          `json:"references,omitempty"`     // posts Message refers to
	Files          []IBFileInfo           `json:"files,omitempty"`          // attached files
	BackReferences []string               `json:"backreferences,omitempty"` // post refering to this post
	Options        map[string]interface{} `json:"options,omitempty"`        // additional stuff
}

// thread in thread list page
type IBThreadListPageThread struct {
	ID                 string       `json:"id"`                  // short ID for references
	OP                 IBPostInfo   `json:"op"`                  // OP
	SkippedReplies     uint32       `json:"skipped_replies"`     // number of replies not included
	SkippedAttachments uint32       `json:"skipped_attachments"` // number of attachments not included
	Replies            []IBPostInfo `json:"replies"`             // replies
}

// info about board common across pages
type IBBoardInfo struct {
	Name        string `json:"name"` // short name
	Description string `json:"desc"` // description
	Info        string `json:"info"` // optional additional info string
}

type IBThreadListPage struct {
	Node     IBNodeInfo               `json:"node"`           // info about this node
	Board    IBBoardInfo              `json:"board"`          // info about this board
	Number   uint32                   `json:"page_number"`    // this page num
	Avaiable uint32                   `json:"pages_avaiable"` // num of pages
	Threads  []IBThreadListPageThread `json:"threads"`        // threads
}

type IBThreadCatalogThread struct {
	ID               string      `json:"id"`                // thread ID
	Thumb            IBThumbInfo `json:"thumb"`             // thumbnail
	TotalReplies     uint32      `json:"total_replies"`     // number of replies
	TotalAttachments uint32      `json:"total_attachments"` // number of attachments
	Subject          string      `json:"subject"`           // subject
	Message          []byte      `json:"message"`           // message
}

type IBThreadCatalog struct {
	Node    IBNodeInfo              `json:"node"`    // info about this node
	Board   IBBoardInfo             `json:"board"`   // info about this baord
	Threads []IBThreadCatalogThread `json:"threads"` // threads
}

type IBThreadPage struct {
	Node    IBNodeInfo   `json:"node"`    // info about this node
	Board   IBBoardInfo  `json:"board"`   // info about this board
	ID      string       `json:"id"`      // thread ID
	OP      IBPostInfo   `json:"op"`      // OP
	Replies []IBPostInfo `json:"replies"` // replies
}
