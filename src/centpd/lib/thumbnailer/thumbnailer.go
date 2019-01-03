package thumbnailer

import (
	"os"

	"centpd/lib/fstore"
	"centpd/lib/ftypes"
)

type ThumbConfig struct {
	Width, Height int    // thumbnail box
	Color         string // background color if needs to be hardcoded
}

type ThumbResult struct {
	// TODO multiple filenames, probably with some attributes each
	FileName      string
	FileExt       string
	Width, Height int
}

type FileInfo struct {
	Kind         ftypes.FTypeT
	DetectedType string
	Attrib       map[string]interface{}
}

type ThumbnailerBuilder interface {
	BuildThumbnailer(fs *fstore.FStore) (Thumbnailer, error)
}

type Thumbnailer interface {
	ThumbProcess(
		f *os.File, ext, mimeType string, cfg ThumbConfig) (
		res ThumbResult, fi FileInfo, err error)
}
