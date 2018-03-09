package tmplrenderer

// simple slow golang' template-based renderer
// renders into html

import (
	. "../logx"
	"../renderer"
	"../webib0"
	"fmt"
	"github.com/BurntSushi/toml"
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

type msgFmtTOML struct {
	PreMsg        string `toml:"pre_msg"`
	PostMsg       string `toml:"post_msg"`
	FirstPreLine  string `toml:"first_pre_line"`
	NextPreLine   string `toml:"next_pre_line"`
	PostLine      string `toml:"post_line"`
	Newline       string `toml:"newline"`
	PreReference  string `toml:"pre_reference"`
	PostReference string `toml:"post_reference"`
}

type msgFmtCfg struct {
	PreMsg       []byte
	PostMsg      []byte
	FirstPreLine []byte
	NextPreLine  []byte
	PostLine     []byte
	Newline      []byte
	preRefTmpl   *template.Template
	postRefTmpl  *template.Template
}

var contentType = "text/html; charset=utf8"

type TmplRenderer struct {
	p webib0.IBProvider
	t [tmplMax]*template.Template
	m msgFmtCfg
	l Logger
}

type TmplRendererCfg struct {
	TemplateDir string
	Logger      LoggerX
}

type NodeInfo struct {
	Name  string
	Root  string
	FRoot string
}

func (tr *TmplRenderer) execTmpl(t int, w io.Writer, d interface{}) {
	err := tr.t[t].Execute(w, d)
	if err != nil {
		tr.l.LogPrintf(ERROR, "%s execution failed: %v", filenames[t], err)
	}
}

func NewTmplRenderer(p webib0.IBProvider, cfg TmplRendererCfg) (*TmplRenderer, error) {
	var err error
	var f []byte
	var t *template.Template

	tr := &TmplRenderer{p: p}
	for i := 0; i < tmplMax; i++ {
		f, err = ioutil.ReadFile(path.Join(cfg.TemplateDir, filenames[i]))
		if err != nil {
			return nil, fmt.Errorf("failed to read %q: %v", filenames[i], err)
		}
		t = template.New(filenames[i]).Funcs(funcs)
		t, err = t.Parse(string(f))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template file %q: %v", filenames[i], err)
		}
		tr.t[i] = t
	}

	f, err = ioutil.ReadFile(path.Join(cfg.TemplateDir, "message.toml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %v", "message.toml", err)
	}
	mtoml := &msgFmtTOML{}
	_, err = toml.Decode(string(f), mtoml)
	if err != nil {
		return nil, fmt.Errorf("failed to parse toml file %q: %v", "message.toml", err)
	}
	tr.m = msgFmtCfg{
		PreMsg:       []byte(mtoml.PreMsg),
		PostMsg:      []byte(mtoml.PostMsg),
		FirstPreLine: []byte(mtoml.FirstPreLine),
		NextPreLine:  []byte(mtoml.NextPreLine),
		PostLine:     []byte(mtoml.PostLine),
		Newline:      []byte(mtoml.Newline),
	}

	t = template.New("pre_reference").Funcs(funcs)
	tr.m.preRefTmpl, err = t.Parse(mtoml.PreReference)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %q: %v", mtoml.PreReference, err)
	}

	t = template.New("post_reference").Funcs(funcs)
	tr.m.postRefTmpl, err = t.Parse(mtoml.PostReference)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %q: %v", mtoml.PostReference, err)
	}

	tr.l = NewLogToX(cfg.Logger, fmt.Sprintf("tmplrenderer.%p", tr))
	return tr, nil
}

func (tr *TmplRenderer) ServeBoardList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentType)

	l := &struct {
		D webib0.IBBoardList
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetBoardList(&l.D)
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

	l := &struct {
		D webib0.IBThreadListPage
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetThreadListPage(&l.D, board, page)
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
	if !l.D.HasBackRefs {
		for i := range l.D.Threads {
			webib0.ProcessBackReferences(&l.D.Threads[i].IBCommonThread)
		}
		l.D.HasBackRefs = true
	}
	tr.execTmpl(tmplThreadListPage, w, l)
}

func (tr *TmplRenderer) ServeThread(w http.ResponseWriter, r *http.Request, board, thread string) {
	w.Header().Set("Content-Type", contentType)

	l := &struct {
		D webib0.IBThreadPage
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetThread(&l.D, board, thread)
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
	if !l.D.HasBackRefs {
		webib0.ProcessBackReferences(&l.D.IBCommonThread)
		l.D.HasBackRefs = true
	}
	tr.execTmpl(tmplThread, w, l)
}

func (tr *TmplRenderer) ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string) {
	w.Header().Set("Content-Type", contentType)

	l := &struct {
		D webib0.IBThreadCatalog
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetThreadCatalog(&l.D, board)
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
