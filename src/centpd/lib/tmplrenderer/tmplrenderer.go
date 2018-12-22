package tmplrenderer

// simple slow golang' template-based renderer
// renders into html

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"strings"
	"text/template"

	. "centpd/lib/logx"
	"centpd/lib/renderer"
	ib0 "centpd/lib/webib0"

	"github.com/BurntSushi/toml"
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
	tmplCreatedBoard
	tmplCreatedBoardErr
	tmplCreatedThread
	tmplCreatedThreadErr
	tmplCreatedPost
	tmplCreatedPostErr
	tmplMax
)

var names = [tmplMax]string{
	"message",
	"board_list",
	"board_list_err",
	"thread_list_page",
	"thread_list_page_err",
	"thread_catalog",
	"thread_catalog_err",
	"thread",
	"thread_err",
	"created_board",
	"created_board_err",
	"created_thread",
	"created_thread_err",
	"created_post",
	"created_post_err",
}

type msgFmtTOML struct {
	PreMsg           string `toml:"pre_msg"`
	PostMsg          string `toml:"post_msg"`
	PreLine          string `toml:"pre_line"`
	PreFirstLine     string `toml:"pre_first_line"`
	PreNonFirstLine  string `toml:"pre_nonfirst_line"`
	PostLine         string `toml:"post_line"`
	PostFinalLine    string `toml:"post_final_line"`
	PostNonFinalLine string `toml:"post_nonfinal_line"`
	Newline          string `toml:"newline"`
	FinalNewline     string `toml:"final_newline"`
	NonFinalNewline  string `toml:"nonfinal_newline"`
	PreQuote         string `toml:"pre_quote"`
	PostQuote        string `toml:"post_quote"`
	PreReference     string `toml:"pre_reference"`
	PostReference    string `toml:"post_reference"`
	TruncationLine   string `toml:"truncation_line"`
}

type msgFmtCfg struct {
	PreMsg  []byte
	PostMsg []byte
	msgLineFmtCfg
	PreQuote           []byte
	PostQuote          []byte
	PreRefTmpl         *template.Template
	PostRefTmpl        *template.Template
	TruncationLineTmpl *template.Template
}

type tmplTOMLSection struct {
	FileName    string `toml:"file"`
	ContentType string `toml:"content_type"`
	Charset     string `toml:"charset"`
}

type tmplTOML map[string]*tmplTOMLSection

type wcCreator func(w http.ResponseWriter) io.WriteCloser

type tmplThing struct {
	t *template.Template // template
	m string             // full mime type
	w wcCreator
}

type TmplRenderer struct {
	p ib0.IBProvider
	t [tmplMax]tmplThing
	m msgFmtCfg
	l Logger
}

type nopWCloser struct {
	io.Writer
}

func (nopWCloser) Close() error {
	return nil
}

func nopWCCreator(w http.ResponseWriter) io.WriteCloser {
	return nopWCloser{w}
}

var _ wcCreator = nopWCCreator

func (tr *TmplRenderer) configTemplates(cfg TmplRendererCfg) error {
	var tt tmplTOML
	var err error

	const tn = "templates.toml"
	cfginfo, err := ioutil.ReadFile(path.Join(cfg.TemplateDir, tn))
	if err != nil {
		tr.l.LogPrintf(INFO, "couldn't read %q: %v", tn, err)
	} else {
		tt = make(tmplTOML)
		err = toml.Unmarshal(cfginfo, &tt)
		if err != nil {
			return fmt.Errorf(
				"failed to parse TOML file %q: %v", tn, err)
		}
	}

	root := template.New("").Funcs(funcs)

	doTemplate := func(name string) (
		t *template.Template, ct string, fe error) {

		filename := name + ".tmpl"
		s, ok := tt[name]
		if ok {
			if s.FileName != "" {
				filename = s.FileName
			}
			ct = s.ContentType
		}

		tinfo, fe := ioutil.ReadFile(path.Join(
			cfg.TemplateDir, filename))
		if fe != nil {
			fe = fmt.Errorf(
				"failed to read template file %q: %v", filename, err)
			return
		}

		t, fe = root.New(name).Parse(string(tinfo))
		if fe != nil {
			fe = fmt.Errorf("failed to parse template file %q: %v",
				filename, fe)
			return
		}

		return
	}

	for i := 0; i < tmplMax; i++ {
		t, ct, e := doTemplate(names[i])
		if e != nil {
			return e
		}

		if ct == "" {
			ct = "text/html"
		}
		mt, par, e := mime.ParseMediaType(ct)
		if e != nil {
			return fmt.Errorf(
				"couldn't parse Content-Type %q: %v", ct, e)
		}
		if strings.HasPrefix(mt, "text/") {
			par["charset"] = "UTF-8"
		}

		tr.t[i].t = t
		tr.t[i].m = mime.FormatMediaType(mt, par)
		tr.t[i].w = nopWCCreator
		delete(tt, names[i])
	}
	for n := range tt {
		_, _, e := doTemplate(n)
		if e != nil {
			return e
		}
	}
	return nil
}

func tmplWC(
	w http.ResponseWriter, tr *TmplRenderer, num int, code int) io.WriteCloser {

	tt := &tr.t[num]
	w.Header().Set("Content-Type", tt.m)
	w.WriteHeader(code)
	return tt.w(w)
}

type TmplRendererCfg struct {
	TemplateDir string
	Logger      LoggerX
}

// TODO utilize this
type NodeInfo struct {
	Name  string
	Root  string
	FRoot string
}

func (tr *TmplRenderer) execTmpl(t int, w io.WriteCloser, d interface{}) {
	err := tr.t[t].t.Execute(w, d)
	if err != nil {
		tr.l.LogPrintf(ERROR, "%s execution failed: %v", names[t], err)
	}
	w.Close()
}

func outTmpl(
	w http.ResponseWriter, tr *TmplRenderer, num int, code int, d interface{}) {

	ww := tmplWC(w, tr, num, code)
	tr.execTmpl(num, ww, d)
}

func (tr *TmplRenderer) configMessage(cfg TmplRendererCfg) error {
	var t *template.Template
	var f []byte
	var err error

	const tn = "message.toml"
	f, err = ioutil.ReadFile(path.Join(cfg.TemplateDir, tn))
	if err != nil {
		return fmt.Errorf("failed to read %q: %v", tn, err)
	}

	mtoml := &msgFmtTOML{}
	_, err = toml.Decode(string(f), mtoml)
	if err != nil {
		tr.l.LogPrintf(ERROR, "failed to parse toml file %q: %v", tn, err)
		return fmt.Errorf("failed to parse toml file %q: %v", tn, err)
	}

	tr.m = msgFmtCfg{
		PreMsg:  []byte(mtoml.PreMsg),
		PostMsg: []byte(mtoml.PostMsg),
		msgLineFmtCfg: msgLineFmtCfg{
			PreFirstLine:     []byte(mtoml.PreFirstLine),
			PreNonFirstLine:  []byte(mtoml.PreNonFirstLine),
			PostFinalLine:    []byte(mtoml.PostFinalLine),
			PostNonFinalLine: []byte(mtoml.PostNonFinalLine),
			FinalNewline:     []byte(mtoml.FinalNewline),
			NonFinalNewline:  []byte(mtoml.NonFinalNewline),
		},
		PreQuote:  []byte(mtoml.PreQuote),
		PostQuote: []byte(mtoml.PostQuote),
	}

	if mtoml.PreFirstLine == "" {
		tr.m.PreFirstLine = []byte(mtoml.PreLine)
	}
	if mtoml.PreNonFirstLine == "" {
		tr.m.PreNonFirstLine = []byte(mtoml.PreLine)
	}

	if mtoml.PostFinalLine == "" {
		tr.m.PostFinalLine = []byte(mtoml.PostLine)
	}
	if mtoml.PostNonFinalLine == "" {
		tr.m.PostNonFinalLine = []byte(mtoml.PostLine)
	}

	if mtoml.FinalNewline == "" {
		tr.m.FinalNewline = []byte(mtoml.Newline)
	}
	if mtoml.NonFinalNewline == "" {
		tr.m.NonFinalNewline = []byte(mtoml.Newline)
	}

	t = template.New("pre_reference").Funcs(funcs)
	tr.m.PreRefTmpl, err = t.Parse(mtoml.PreReference)
	if err != nil {
		return fmt.Errorf("failed to parse template %q: %v",
			mtoml.PreReference, err)
	}

	t = template.New("post_reference").Funcs(funcs)
	tr.m.PostRefTmpl, err = t.Parse(mtoml.PostReference)
	if err != nil {
		return fmt.Errorf("failed to parse template %q: %v",
			mtoml.PostReference, err)
	}

	t = template.New("truncation_line").Funcs(funcs)
	tr.m.TruncationLineTmpl, err = t.Parse(mtoml.TruncationLine)
	if err != nil {
		return fmt.Errorf("failed to parse template %q: %v",
			mtoml.TruncationLine, err)
	}

	return nil
}

func NewTmplRenderer(
	p ib0.IBProvider, cfg TmplRendererCfg) (*TmplRenderer, error) {

	var err error

	tr := &TmplRenderer{p: p}

	tr.l = NewLogToX(cfg.Logger, fmt.Sprintf("tmplrenderer.%p", tr))

	err = tr.configTemplates(cfg)
	if err != nil {
		return nil, err
	}

	err = tr.configMessage(cfg)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

func (tr *TmplRenderer) ServeBoardList(
	w http.ResponseWriter, r *http.Request) {

	l := &struct {
		D ib0.IBBoardList
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetBoardList(&l.D)
	if err != nil {
		ctx := struct {
			Code int
			Err  error
		}{
			code,
			err,
		}
		outTmpl(w, tr, tmplBoardListErr, code, ctx)
		return
	}
	outTmpl(w, tr, tmplBoardList, 200, l)
}

func (tr *TmplRenderer) ServeThreadListPage(
	w http.ResponseWriter, r *http.Request, board string, page uint32) {

	l := &struct {
		D ib0.IBThreadListPage
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetThreadListPage(&l.D, board, page)
	if err != nil {
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
		outTmpl(w, tr, tmplThreadListPageErr, code, ctx)
		return
	}
	if !l.D.HasBackRefs {
		for i := range l.D.Threads {
			ib0.ProcessBackReferences(&l.D.Threads[i].IBCommonThread)
		}
		l.D.HasBackRefs = true
	}
	outTmpl(w, tr, tmplThreadListPage, 200, l)
}

func (tr *TmplRenderer) ServeThread(
	w http.ResponseWriter, r *http.Request, board, thread string) {

	l := &struct {
		D ib0.IBThreadPage
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetThread(&l.D, board, thread)
	if err != nil {
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
		outTmpl(w, tr, tmplThreadErr, code, ctx)
		return
	}
	if !l.D.HasBackRefs {
		ib0.ProcessBackReferences(&l.D.IBCommonThread)
		l.D.HasBackRefs = true
	}
	outTmpl(w, tr, tmplThread, 200, l)
}

func (tr *TmplRenderer) ServeThreadCatalog(
	w http.ResponseWriter, r *http.Request, board string) {

	l := &struct {
		D ib0.IBThreadCatalog
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetThreadCatalog(&l.D, board)
	if err != nil {
		ctx := struct {
			Code  int
			Err   error
			Board string
		}{
			code,
			err,
			board,
		}
		outTmpl(w, tr, tmplThreadCatalogErr, code, ctx)
		return
	}
	outTmpl(w, tr, tmplThreadCatalog, 200, l)
}

func (tr *TmplRenderer) DressNewBoardResult(
	w http.ResponseWriter, bname string, err error, code int) {

	l := &struct {
		S bool   // success
		B string // board name
		E error
		C int
		N NodeInfo
		R *TmplRenderer
	}{
		S: err == nil,
		B: bname,
		E: err,
		C: code,
		R: tr,
	}
	if err == nil {
		outTmpl(w, tr, tmplCreatedBoard, 200, l)
	} else {
		outTmpl(w, tr, tmplCreatedBoardErr, code, l)
	}
}

func (tr *TmplRenderer) DressPostResult(
	w http.ResponseWriter, pi ib0.IBPostedInfo, newthread bool,
	err error, code int) {

	l := &struct {
		D ib0.IBPostedInfo
		E error
		C int
		N NodeInfo
		R *TmplRenderer
	}{
		D: pi,
		E: err,
		C: code,
		R: tr,
	}
	if newthread {
		if err == nil {
			outTmpl(w, tr, tmplCreatedThread, 200, l)
		} else {
			outTmpl(w, tr, tmplCreatedThreadErr, code, l)
		}
	} else {
		if err == nil {
			outTmpl(w, tr, tmplCreatedPost, 200, l)
		} else {
			outTmpl(w, tr, tmplCreatedPostErr, code, l)
		}
	}
}
