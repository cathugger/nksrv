package ftypes

type FTypeT int

const (
	FTypeFile FTypeT = iota
	FTypeMsg
	FTypeText
	FTypeImage
)

var FTypeS = map[FTypeT]string{
	FTypeFile:  "file",
	FTypeMsg:   "msg",
	FTypeText:  "text",
	FTypeImage: "image",
}
