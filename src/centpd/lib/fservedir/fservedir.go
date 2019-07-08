package fservedir

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"centpd/lib/emime"
	"centpd/lib/fserve"
)

var _ fserve.FServe = (*FServeDir)(nil)

type MIMETypeSetter interface {
	SetMIMETypeByName(w http.ResponseWriter, name string)
}

type FServeDir struct {
	dir         string
	cachectlstr string
	mts         MIMETypeSetter
}

type DefaultMIMETypeSetter struct{}

func (DefaultMIMETypeSetter) SetMIMETypeByName(w http.ResponseWriter, name string) {
	var mimeType string
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		mimeType = emime.MIMETypeByExtension(name[i+1:])
	} else {
		mimeType = emime.MIMETypeByExtension("")
	}
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
}

type Config struct {
	MIMETypeSetter MIMETypeSetter
	CacheControl   string
}

func NewFServeDir(dir string, cfg Config) *FServeDir {
	if i := len(dir); i > 0 && !os.IsPathSeparator(dir[i-1]) {
		dir = dir + string(os.PathSeparator)
	}

	if cfg.MIMETypeSetter == nil {
		cfg.MIMETypeSetter = DefaultMIMETypeSetter{}
	}

	return &FServeDir{
		dir:         dir,
		mts:         cfg.MIMETypeSetter,
		cachectlstr: cfg.CacheControl,
	}
}

func (d *FServeDir) FServe(w http.ResponseWriter, r *http.Request, id string) {

	fname := d.dir + id

	sl := strings.LastIndexByte(id, '/')
	if sl >= 0 {
		id = id[sl+1:]
	}
	if id == "" {
		http.NotFound(w, r)
		return
	}

	f, err := os.Open(fname)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	// set Content-Type and whatever else
	d.mts.SetMIMETypeByName(w, id)

	if d.cachectlstr != "" {
		w.Header().Set("Cache-Control", d.cachectlstr)
	}

	fi, err := f.Stat()
	if err == nil {
		http.ServeContent(w, r, fname, fi.ModTime(), f)
	} else {
		// shouldn't happen
		//http.ServeContent(w, r, fname, time.Time{}, f)
		//panic(fmt.Errorf("f.Stat() failed: %v", err))
		s := fmt.Sprintf(
			"500 internal server error: failed to stat %q: %v", fname, err)
		http.Error(w, s, http.StatusInternalServerError)
	}
}

func (d *FServeDir) Dir() string {
	return d.dir
}
