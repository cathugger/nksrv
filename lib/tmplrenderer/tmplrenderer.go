package tmplrenderer

// simple slow golang' template-based renderer
// renders into html

import (
	"../renderer"
	"../webib0"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"
)

var _ renderer.Renderer = (*TmplRenderer)(nil)

const (
	tmplBoardList = iota
	tmplBoardListErr
	tmplThreadListPage
	tmplThreadListPageErr
	tmplThreadCatalog
	tmplThreadCatalogErr
	tmplThread
	tmplThreadErr
	tmplMax
)

var filenames = [tmplMax]string{
	"board_list.tmpl",
	"board_list_err.tmpl",
	"thread_list_page.tmpl",
	"thread_list_page_err.tmpl",
	"thread_catalog.tmpl",
	"thread_catalog_err.tmpl",
	"thread.tmpl",
	"thread_err.tmpl",
}

var contentType = "text/html; charset=utf8"

var funcs = map[string]interface{}{
	"urlpath":    urlPath,
	"truncatefn": truncatefn,
	"filesize":   filesize,
	"date":       date,
}

func urlPath(p string) string {
	return (&url.URL{Path: p}).EscapedPath()
}

func truncatefn(s string, l int) string {
	if utf8.RuneCountInString(s) <= l {
		// fast path, no truncation needed
		return s
	}
	i := strings.LastIndexByte(s, '.')
	// assume extension isnt special snowflake utf8
	// if there is no dot or len("(...).ext") would exceed our limits
	if i < 0 || 5+(len(s)-i) > l {
		// use "filename..." form instead which doesnt give special treatment to extension
		canuse := l - 3
		x, j := 0, 0
		for j = range s {
			if x >= canuse {
				break
			}
			x++
		}
		return s[:j] + "..."
	}
	// use "fn(...).ext" form
	canuse := l - 5 - (len(s) - i)
	x, j := 0, 0
	for j = range s {
		if x >= canuse {
			break
		}
		x++
	}
	return s[:j] + "(...)" + s[i:]
}

func filesize(s int64) string {
	if s < 1<<10 {
		return fmt.Sprintf("%d B", s)
	}
	if s < 1<<20 {
		return fmt.Sprintf("%.3f KiB", float64(s)/(1<<10))
	}
	if s < 1<<30 {
		return fmt.Sprintf("%.3f MiB", float64(s)/(1<<20))
	}
	if s < 1<<40 {
		return fmt.Sprintf("%.3f GiB", float64(s)/(1<<30))
	}
	return fmt.Sprintf("%.6f TiB", float64(s)/(1<<40))
}

func date(u int64) string {
	t := time.Unix(u, 0)
	Y, M, D := t.Date()
	h, m, s := t.Hour(), t.Minute(), t.Second()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", Y, M, D, h, m, s)
}

type TmplRenderer struct {
	p webib0.IBProvider
	t [tmplMax]*template.Template
}

func (tr *TmplRenderer) execTmpl(t int, w io.Writer, d interface{}) {
	err := tr.t[t].Execute(w, d)
	if err != nil {
		// XXX better logger
		fmt.Fprintf(os.Stderr, "%s execution failed: %v\n", filenames[t], err)
	}
}

func NewTmplRenderer(p webib0.IBProvider, tdir string) (*TmplRenderer, error) {
	var err error
	tr := &TmplRenderer{p: p}
	for i := 0; i < tmplMax; i++ {
		var f []byte
		f, err = ioutil.ReadFile(path.Join(tdir, filenames[i]))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %v", filenames[i], err)
		}
		t := template.New(filenames[i]).Funcs(funcs)
		t, err = t.Parse(string(f))
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %v", filenames[i], err)
		}
		tr.t[i] = t
	}
	return tr, nil
}

func (tr *TmplRenderer) ServeBoardList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentType)
	l := &webib0.IBBoardList{}
	err, code := tr.p.IBGetBoardList(l)
	if err != nil {
		w.WriteHeader(code)
		ctx := struct {
			Code int
			Err  error
		}{
			code,
			err,
		}
		tr.execTmpl(tmplBoardListErr, w, ctx)
		return
	}
	tr.execTmpl(tmplBoardList, w, l)
}

func (tr *TmplRenderer) ServeThreadListPage(w http.ResponseWriter, r *http.Request, board string, page uint32) {
	w.Header().Set("Content-Type", contentType)
	l := &webib0.IBThreadListPage{}
	err, code := tr.p.IBGetThreadListPage(l, board, page)
	if err != nil {
		w.WriteHeader(code)
		ctx := struct {
			Code  int
			Err   error
			Board string
			Page  uint32
		}{
			code,
			err,
			board,
			page,
		}
		tr.execTmpl(tmplThreadListPageErr, w, ctx)
		return
	}
	tr.execTmpl(tmplThreadListPage, w, l)
}

func (tr *TmplRenderer) ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string) {
	w.Header().Set("Content-Type", contentType)
	l := &webib0.IBThreadCatalog{}
	err, code := tr.p.IBGetThreadCatalog(l, board)
	if err != nil {
		w.WriteHeader(code)
		ctx := struct {
			Code  int
			Err   error
			Board string
		}{
			code,
			err,
			board,
		}
		tr.execTmpl(tmplThreadCatalogErr, w, ctx)
		return
	}
	tr.execTmpl(tmplThreadCatalog, w, l)
}

func (tr *TmplRenderer) ServeThread(w http.ResponseWriter, r *http.Request, board, thread string) {
	w.Header().Set("Content-Type", contentType)
	l := &webib0.IBThreadPage{}
	err, code := tr.p.IBGetThread(l, board, thread)
	if err != nil {
		w.WriteHeader(code)
		ctx := struct {
			Code   int
			Err    error
			Board  string
			Thread string
		}{
			code,
			err,
			board,
			thread,
		}
		tr.execTmpl(tmplThreadErr, w, ctx)
		return
	}
	tr.execTmpl(tmplThread, w, l)
}
