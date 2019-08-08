package ftypes

type FTypeT int

const (
	FTypeFile FTypeT = iota
	FTypeMsg
	FTypeFace
	FTypeText
	FTypeImage
	FTypeAudio
	FTypeVideo
	_FTypeMax
)

var FTypeS = [_FTypeMax]string{
	FTypeFile:  "file",
	FTypeMsg:   "msg",
	FTypeFace:  "face",
	FTypeText:  "text",
	FTypeImage: "image",
	FTypeAudio: "audio",
	FTypeVideo: "video",
}

var FTypeM map[string]FTypeT

func init() {
	FTypeM = make(map[string]FTypeT)
	for i, v := range FTypeS {
		FTypeM[v] = FTypeT(i)
	}
}

func StringToFType(s string) FTypeT {
	return FTypeM[s]
}

func (t FTypeT) String() string {
	return FTypeS[t]
}

// vague term, but means that it's actual attachment, not some nksrv-internal thing
func (t FTypeT) Normal() bool {
	return t != FTypeMsg && t != FTypeFace
}
