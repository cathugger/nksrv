package psqlib

import ib0 "centpd/lib/webib0"

type boardID = uint32

type postID = uint64

// structures

type submissionLimits struct {
	// message stuff
	MaxTitleLength   uint32 `json:"title_max_size,omitempty"`
	MaxNameLength    uint32 `json:"name_max_size,omitempty"`
	MaxMessageLength uint32 `json:"message_max_size,omitempty"`

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
	MaxTitleLength:   64,
	MaxNameLength:    64,
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
	Info string   `json:"info,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

var defaultBoardAttributes = boardAttributes{}

type threadOptions struct {
	///Locked     bool   `json:"locked,omitempty"`     // do not allow non-mod posts? or any posts at all??
	///PostLimit  uint32 `json:"post_limit,omitempty"` // do not bump after thread has this much posts. is this behavior good?
	BumpLimit uint32 `json:"bump_limit,omitempty"` // do not bump after thread has this much (non-sage or sage, doesn't matter) posts
	//FileLimit uint32 `json:"file_limit,omitempty"` // TODO decide should we count OP files or not. 4chan doesnt count but it always forces single file on OP. should investigate other imageboards.
	// 8ch does not count OP images, and counts every post with files (files aren't distinctly counted, just whether post has file or not)
	// from sum-of-all-files-sizes limit perspective this makes sense I guess
}

var defaultThreadOptions = threadOptions{
	BumpLimit: 300,
	//FileLimit: 150,
}

type postAttributes struct {
	References []ib0.IBMessageReference `json:"refs,omitempty"`
}

var defaultPostAttributes = postAttributes{}

type thumbAttributes struct {
	Width  uint32 `json:"w"`
	Height uint32 `json:"h"`
}

var defaultThumbAttributes = thumbAttributes{}
