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
	FileMinNum        uint32 `json:"file_min_num,omitempty"`
	FileMaxNum        uint32 `json:"file_max_num,omitempty"`    // 0 - unlimited
	FileMaxSizeSingle uint64 `json:"file_max_single,omitempty"` // 0 - unlimited
	FileMaxSizeAll    uint64 `json:"file_max_all,omitempty"`    // 0 - unlimited

	// file extensions. these do NOT include dot. empty extension should match both "file" and "file."
	ExtWhitelist bool     `json:"ext_whitelist,omitempty"` // list mode. defaults to false which means blacklist
	ExtAllow     []string `json:"ext_allow,omitempty"`     // whitelist
	ExtDeny      []string `json:"ext_deny,omitempty"`      // blacklist
}

var defaultReplySubmissionLimits = submissionLimits{
	MaxTitleLength:   48,
	MaxMessageLength: 2000,

	FileMaxNum:     5,
	FileMaxSizeAll: 8 * 1024 * 1024,
}

// :^)
var defaultNewThreadSubmissionLimits = func(l submissionLimits) submissionLimits {
	l.FileMinNum = 1
	return l
}(defaultReplySubmissionLimits)

type boardAttributes struct {
	Description string   `json:"desc,omitempty"`
	Info        string   `json:"info,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

var defaultBoardAttributes = boardAttributes{}

type threadOptions struct {
	///Locked     bool   `json:"locked,omitempty"`     // do not allow non-mod posts? or any posts at all??
	///PostLimit  uint32 `json:"post_limit,omitempty"` // do not bump after thread has this much posts. is this behavior good?
	BumpLimit uint32 `json:"bump_limit,omitempty"` // do not bump after thread has this much non-sage posts
	FileLimit uint32 `json:"file_limit,omitempty"`
}

var defaultThreadOptions = threadOptions{
	BumpLimit: 300,
	FileLimit: 150,
}

type threadAttributes struct {
}

var defaultThreadAttributes = threadAttributes{}

type postAttributes struct {
	References []ib0.IBMessageReference `json:"refs,omitempty"`
}

var defaultPostAttributes = postAttributes{}

type thumbAttributes struct {
	Width  uint32 `json:"w"`
	Height uint32 `json:"h"`
}

var defaultThumbAttributes = thumbAttributes{}
