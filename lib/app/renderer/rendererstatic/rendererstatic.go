package rendererstatic

// not really renderer, just for html testing

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"nksrv/lib/app/renderer"
)

var _ renderer.Renderer = RendererStatic{}

type RendererStatic struct{}

const contentType = "text/html; charset=UTF-8"
const serveDir = "_demo/statichtml/"

func doServe(w http.ResponseWriter, r *http.Request, s string) {
	fname := serveDir + s
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

func (RendererStatic) ServeBoardList(w http.ResponseWriter, r *http.Request) {
	doServe(w, r, "boardlist.html")
}

func (RendererStatic) ServeThreadListPage(w http.ResponseWriter, r *http.Request, board string, page uint32) {
	doServe(w, r, "b-"+board+"-"+strconv.Itoa(int(page))+".html")
}

func (RendererStatic) ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string) {
	doServe(w, r, "c-"+board+".html")
}

func (RendererStatic) ServeThread(w http.ResponseWriter, r *http.Request, board, thread string) {
	doServe(w, r, "t-"+board+"-"+thread+".html")
}
