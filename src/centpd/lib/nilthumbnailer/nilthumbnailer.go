package nilthumbnailer

import (
	"os"

	"centpd/lib/thumbnailer"
)

type NilThumbnailer struct{}

func (NilThumbnailer) ThumbProcess(
	f *os.File, ext, mimeType string, cfg thumbnailer.ThumbConfig) (
	res thumbnailer.ThumbResult, fi thumbnailer.FileInfo, err error) {

	err = f.Close()
	return
}

var _ thumbnailer.Thumbnailer = NilThumbnailer{}
