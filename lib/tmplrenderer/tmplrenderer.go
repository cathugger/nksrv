package tmplrenderer

// simple slow golang' template-based renderer
// renders into html

import (
	. "../logx"
	"../renderer"
	"../webib0"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"text/template"
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

type TmplRenderer struct {
	p webib0.IBProvider
	t [tmplMax]*template.Template
	l Logger
}

type TmplRendererCfg struct {
	TemplateDir string
	Logger      LoggerX
}

func (tr *TmplRenderer) execTmpl(t int, w io.Writer, d interface{}) {
	err := tr.t[t].Execute(w, d)
	if err != nil {
		tr.l.LogPrintf(ERROR, "%s execution failed: %v", filenames[t], err)
	}
}

func NewTmplRenderer(p webib0.IBProvider, cfg TmplRendererCfg) (*TmplRenderer, error) {
	var err error
	tr := &TmplRenderer{p: p}
	for i := 0; i < tmplMax; i++ {
		var f []byte
		f, err = ioutil.ReadFile(path.Join(cfg.TemplateDir, filenames[i]))
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
	tr.l = NewLogToX(cfg.Logger, fmt.Sprintf("tmplrenderer.%p", tr))
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
