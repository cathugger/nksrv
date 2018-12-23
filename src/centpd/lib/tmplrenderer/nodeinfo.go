package tmplrenderer

// TODO utilize this
type NodeInfo struct {
	Name  string
	Root  string // for web serving, static, and API
	FRoot string // for file serving
	PRoot string // for post submission
}
