package rendererstatic

// not really renderer, just for html testing

import (
	"nekochan/lib/renderer"
	"net/http"
	"os"
	"strconv"
	"time"
)

var _ renderer.Renderer = RendererStatic{}

type RendererStatic struct{}

const contentType = "text/html; charset=utf8"
const serveDir = "_demo/statichtml/"

func (RendererStatic) ServeBoardList(w http.ResponseWriter, r *http.Request) {
	fname := serveDir + "boardlist.html"
	f, err := os.Open(fname)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", contentType)

	if fi, err := f.Stat(); err == nil {
		http.ServeContent(w, r, fname, fi.ModTime(), f)
	} else {
		http.ServeContent(w, r, fname, time.Time{}, f)
	}
}

func (RendererStatic) ServeThreadListPage(w http.ResponseWriter, r *http.Request, board string, page uint32) {
	fname := serveDir + "b-" + board + "-" + strconv.Itoa(int(page)) + ".html"
	f, err := os.Open(fname)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", contentType)

	if fi, err := f.Stat(); err == nil {
		http.ServeContent(w, r, fname, fi.ModTime(), f)
	} else {
		http.ServeContent(w, r, fname, time.Time{}, f)
	}
}

func (RendererStatic) ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string) {
	fname := serveDir + "c-" + board + ".html"
	f, err := os.Open(fname)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", contentType)

	if fi, err := f.Stat(); err == nil {
		http.ServeContent(w, r, fname, fi.ModTime(), f)
	} else {
		http.ServeContent(w, r, fname, time.Time{}, f)
	}
}

func (RendererStatic) ServeThread(w http.ResponseWriter, r *http.Request, board, thread string) {
	fname := serveDir + "t-" + board + "-" + thread + ".html"
	f, err := os.Open(fname)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", contentType)

	if fi, err := f.Stat(); err == nil {
		http.ServeContent(w, r, fname, fi.ModTime(), f)
	} else {
		http.ServeContent(w, r, fname, time.Time{}, f)
	}
}
