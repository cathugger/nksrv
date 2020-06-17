package pibase

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
