package nilthm

import (
	"os"

	"nksrv/lib/thumbnailer"
)

type NilThumbnailer struct{}

func (NilThumbnailer) ThumbProcess(
	f *os.File, ext, mimeType string, fsize int64, cfg thumbnailer.ThumbConfig) (
	res thumbnailer.ThumbResult, err error) {

	err = f.Close()
	return
}

var _ thumbnailer.Thumbnailer = NilThumbnailer{}
