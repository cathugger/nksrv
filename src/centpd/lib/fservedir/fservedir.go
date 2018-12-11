package fservedir

import (
	"net/http"
	"os"
	"strings"
	"time"

	"centpd/lib/emime"
	"centpd/lib/fserve"
)

var _ fserve.FServe = FServeDir{}

type FServeDir struct {
	dir string
}

func NewFServeDir(dir string) FServeDir {
	i := len(dir)
	if i > 0 && !os.IsPathSeparator(dir[i-1]) {
		dir = dir + string(os.PathSeparator)
	}
	return FServeDir{dir: dir}
}

func (d FServeDir) FServe(w http.ResponseWriter, r *http.Request, id string) {
	fname := d.dir + id

	f, err := os.Open(fname)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	var mimeType string
	if i := strings.LastIndexByte(id, '.'); i >= 0 {
		mimeType = emime.MIMETypeByExtension(id[i+1:])
	} else {
		mimeType = emime.MIMETypeByExtension("")
	}
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}

	fi, err := f.Stat()
	if err == nil {
		http.ServeContent(w, r, fname, fi.ModTime(), f)
	} else {
		// shouldn't happen
		http.ServeContent(w, r, fname, time.Time{}, f)
	}
}
