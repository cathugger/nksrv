package ibattribs

type BoardAttribs struct {
	Info string   `json:"info,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

var DefaultBoardAttribs = BoardAttribs{}

type BoardPostAttribs struct{}

var DefaultBoardPostAttribs = BoardPostAttribs{}

type GlobalPostAttribs struct{}

var DefaultGlobalPostAttribs = GlobalPostAttribs{}

type ThumbAttribs struct {
	Width  uint32 `json:"w,omitempty"`
	Height uint32 `json:"h,omitempty"`
}

var DefaultThumbAttribs = ThumbAttribs{}
