package psqlib

import ib0 "nekochan/lib/webib0"

type boardID = uint32

type postID = uint64

// structures

type submissionLimits struct {
	// message stuff
	MaxTitleLength   uint32 `json:"max_len_title,omitempty"`
	MaxMessageLength uint32 `json:"max_len_msg,omitempty"`

	// files count and sizes
	FileMaxNum        uint32 `json:"file_max_num,omitempty"`    // 0 - unlimited
	FileMaxSizeSingle uint64 `json:"file_max_single,omitempty"` // 0 - unlimited
	FileMaxSizeAll    uint64 `json:"file_max_all,omitempty"`    // 0 - unlimited

	// file extensions. these do NOT include dot. empty extension should match both "file" and "file."
	ExtWhitelist bool     `json:"ext_whitelist,omitempty"` // list mode. defaults to false which means blacklist
	ExtAllow     []string `json:"ext_allow,omitempty"`     // whitelist
	ExtDeny      []string `json:"ext_deny,omitempty"`      // blacklist
}

type boardAttributes struct {
	Description    string           `json:"desc,omitempty"`
	Info           string           `json:"info,omitempty"`
	Tags           []string         `json:"tags,omitempty"`
	PageLimit      uint32           `json:"page_limit,omitempty"`
	ThreadsPerPage uint32           `json:"threads_per_page,omitempty"`
	ThreadLimits   submissionLimits `json:"threadlimits,omitempty"`
}

var defaultBoardAttributes = boardAttributes{
	ThreadsPerPage: 10,
	ThreadLimits: submissionLimits{
		MaxTitleLength:   48,
		MaxMessageLength: 2000,
	},
}

type threadAttributes struct {
	Locked      bool             `json:"locked,omitempty"`
	ReplyLimits submissionLimits `json:"replylimits,omitempty"`
}

var defaultThreadAttributes = threadAttributes{}

type postAttributes struct {
	References []ib0.IBMessageReference `json:"refs"`
}

var defaultPostAttributes = postAttributes{}

type thumbAttributes struct {
	Width  uint32 `json:"w"`
	Height uint32 `json:"h"`
}

var defaultThumbAttributes = thumbAttributes{}
