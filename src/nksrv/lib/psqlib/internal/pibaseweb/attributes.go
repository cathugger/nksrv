package psqlib

import "nksrv/lib/ibattribs"

type TBoardID = uint32
type TPostID = uint64

// structures

type SubmissionLimits struct {
	// message stuff
	MaxTitleLength   uint32 `json:"title_max_size,omitempty"`
	MaxNameLength    uint32 `json:"name_max_size,omitempty"`
	MaxMessageLength uint32 `json:"message_max_size,omitempty"`

	// files count and sizes
	FileMinNum        int32 `json:"file_min_num,omitempty"`    // <= 0 - no minimum
	FileMaxNum        int32 `json:"file_max_num,omitempty"`    // <= 0 - don't allow attach (no upper limit makes no sense)
	FileMaxSizeSingle int64 `json:"file_max_single,omitempty"` // <= 0 - unlimited
	FileMaxSizeAll    int64 `json:"file_max_all,omitempty"`    // <= 0 - unlimited

	// file extensions. these do NOT include dot. empty extension should match both "file" and "file."
	ExtWhitelist bool     `json:"ext_whitelist,omitempty"` // list mode. defaults to false which means blacklist
	ExtAllow     []string `json:"ext_allow,omitempty"`     // whitelist
	ExtDeny      []string `json:"ext_deny,omitempty"`      // blacklist
}

var DefaultReplySubmissionLimits = SubmissionLimits{
	MaxTitleLength:   64,
	MaxNameLength:    64,
	MaxMessageLength: 8000,

	FileMaxNum:     5,
	FileMaxSizeAll: 8 * 1024 * 1024,
}

var MaxSubmissionLimits = SubmissionLimits{FileMaxNum: 0x7FffFFff}

// :^)
var DefaultNewThreadSubmissionLimits = func(l SubmissionLimits) SubmissionLimits {
	l.FileMinNum = 1
	return l
}(DefaultReplySubmissionLimits)

type ThreadOptions struct {
	///Locked     bool   `json:"locked,omitempty"`     // do not allow non-mod posts? or any posts at all??
	///PostLimit  uint32 `json:"post_limit,omitempty"` // do not bump after thread has this much posts. is this behavior good?
	BumpLimit uint32 `json:"bump_limit,omitempty"` // do not bump after thread has this much (non-sage or sage, doesn't matter) posts
	FileLimit uint32 `json:"file_limit,omitempty"`
	// XXX decide should we count OP files or not. 4chan doesnt count but it always forces single file on OP.
	// 8ch does not count OP images, and counts every post with files (files aren't distinctly counted, just whether post has file or not)
	// from sum-of-all-files-sizes limit perspective this makes sense I guess
	// current stuff won't count files individually and won't count OP
}

var DefaultThreadOptions = ThreadOptions{
	BumpLimit: 300,
	FileLimit: 150,
}

var DefaultBoardAttributes = ibattribs.DefaultBoardAttribs
var DefaultThumbAttributes = ibattribs.DefaultThumbAttribs
