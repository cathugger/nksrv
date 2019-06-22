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

// page
const (
	ptmplBoardList = iota
	ptmplBoardListErr
	ptmplThreadListPage
	ptmplThreadListPageErr
	ptmplOverboardPage
	ptmplOverboardPageErr
	ptmplThreadCatalog
	ptmplThreadCatalogErr
	ptmplThread
	ptmplThreadErr

	ptmplMax
)

// result
const (
	rtmplCreatedBoard = iota
	rtmplCreatedBoardErr
	rtmplCreatedThread
	rtmplCreatedThreadErr
	rtmplCreatedPost
	rtmplCreatedPostErr

	rtmplMax
)

var pnames = [ptmplMax]string{
	"board_list",
	"board_list_err",
	"thread_list_page",
	"thread_list_page_err",
	"overboard_page",
	"overboard_page_err",
	"thread_catalog",
	"thread_catalog_err",
	"thread",
	"thread_err",
}
var rnames = [rtmplMax]string{
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
	PreMsg    []byte
	PostMsg   []byte
	PreQuote  []byte
	PostQuote []byte

	msgLineFmtCfg

	PreRefTmpl         *template.Template
	PostRefTmpl        *template.Template
	TruncationLineTmpl *template.Template
}

type tmplTOMLSection struct {
	ContentType string `toml:"content_type"`
	Base        string `toml:"base"`
}

type tmplTOML map[string]*tmplTOMLSection

type wcCreator func(w http.ResponseWriter) io.WriteCloser

type tmplThing struct {
	t *template.Template // template
	m string             // full mime type
	w wcCreator
}

type TmplRenderer struct {
	p  ib0.IBProvider
	tp [ptmplMax]tmplThing
	tr [rtmplMax]tmplThing
	m  msgFmtCfg
	l  Logger
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

	const tn = "templates.toml"
	cfginfo, xe := ioutil.ReadFile(path.Join(cfg.TemplateDir, tn))
	if xe != nil {
		tr.l.LogPrintf(INFO, "couldn't read %q: %v", tn, xe)
	} else {
		tt = make(tmplTOML)
		e := toml.Unmarshal(cfginfo, &tt)
		if e != nil {
			return fmt.Errorf(
				"failed to parse TOML file %q: %v", tn, e)
		}
	}

	root := template.New("").Funcs(funcs)
	mc := metaContext{
		dir:         cfg.TemplateDir,
		captchamode: "",
	}

	doTemplate := func(base, name string) (
		t *template.Template, ct string, fe error) {

		s, ok := tt[name]
		if ok {
			ct = s.ContentType
			if s.Base != "" {
				base = s.Base
			}
		}

		tinfo, fe := loadMetaTmpl(mc, base, name)
		if fe != nil {
			fe = fmt.Errorf(
				"failed to read template %q file: %v", name, fe)
			return
		}

		t, fe = root.New(name).Parse(string(tinfo))
		if fe != nil {
			fe = fmt.Errorf(
				"failed to parse template %q file: %v", name, fe)
			return
		}

		return
	}
	doFullTemplate := func(base, name string) (
		t *template.Template, ft string, fe error) {

		t, ct, fe := doTemplate(base, name)
		if fe != nil {
			return
		}
		if ct == "" {
			// TODO configurable
			ct = "text/html"
		}
		mt, par, fe := mime.ParseMediaType(ct)
		if fe != nil {
			fe = fmt.Errorf("couldn't parse Content-Type %q: %v", ct, fe)
			return
		}
		if strings.HasPrefix(mt, "text/") {
			par["charset"] = "UTF-8"
		}

		return t, mime.FormatMediaType(mt, par), nil
	}
	for i := range pnames {
		t, ct, e := doFullTemplate("", pnames[i])
		if e != nil {
			return e
		}
		tr.tp[i].t = t
		tr.tp[i].m = ct
		tr.tp[i].w = nopWCCreator
		delete(tt, pnames[i])
	}
	for i := range rnames {
		t, ct, e := doFullTemplate("", rnames[i])
		if e != nil {
			return e
		}
		tr.tr[i].t = t
		tr.tr[i].m = ct
		tr.tr[i].w = nopWCCreator
		delete(tt, rnames[i])
	}
	for n := range tt {
		_, _, e := doTemplate("", n)
		if e != nil {
			return e
		}
	}
	return nil
}

func tmplWC(
	w http.ResponseWriter, tt *tmplThing, code int) io.WriteCloser {

	w.Header().Set("Content-Type", tt.m)
	w.WriteHeader(code)
	return tt.w(w)
}

type TmplRendererCfg struct {
	TemplateDir string
	Logger      LoggerX
}

func (tr *TmplRenderer) execTmpl(
	tt *tmplThing, tname string, w io.WriteCloser, d interface{}) {

	err := tt.t.Execute(w, d)
	if err != nil {
		tr.l.LogPrintf(ERROR, "%s execution failed: %v", tname, err)
	}
	w.Close()
}

func (tr *TmplRenderer) outTmplX(
	w http.ResponseWriter, tt *tmplThing, tname string, code int, d interface{}) {

	ww := tmplWC(w, tt, code)
	tr.execTmpl(tt, tname, ww, d)
}

func (tr *TmplRenderer) outTmplP(
	w http.ResponseWriter, tid, code int, d interface{}) {

	tr.outTmplX(w, &tr.tp[tid], pnames[tid], code, d)
}

func (tr *TmplRenderer) outTmplR(
	w http.ResponseWriter, tid, code int, d interface{}) {

	tr.outTmplX(w, &tr.tr[tid], rnames[tid], code, d)
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
	p ib0.IBProvider, cfg TmplRendererCfg) (tr *TmplRenderer, err error) {

	tr = &TmplRenderer{p: p}

	tr.l = NewLogToX(cfg.Logger, fmt.Sprintf("tmplrenderer.%p", tr))

	err = tr.configTemplates(cfg)
	if err != nil {
		return
	}

	err = tr.configMessage(cfg)
	if err != nil {
		return
	}

	return
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
		tr.outTmplP(w, ptmplBoardListErr, code, ctx)
		return
	}
	tr.outTmplP(w, ptmplBoardList, 200, l)
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
		tr.outTmplP(w, ptmplThreadListPageErr, code, ctx)
		return
	}
	if !l.D.HasBackRefs {
		for i := range l.D.Threads {
			ib0.ProcessBackReferences(l.D.Board.Name, &l.D.Threads[i].IBCommonThread)
		}
		l.D.HasBackRefs = true
	}
	tr.outTmplP(w, ptmplThreadListPage, 200, l)
}

func (tr *TmplRenderer) ServeOverboardPage(
	w http.ResponseWriter, r *http.Request, page uint32) {

	l := &struct {
		D ib0.IBOverboardPage
		N NodeInfo
		R *TmplRenderer
	}{
		R: tr,
	}

	err, code := tr.p.IBGetOverboardPage(&l.D, page)
	if err != nil {
		ctx := struct {
			Code int
			Err  error
			Page uint32
		}{
			code,
			err,
			page,
		}
		tr.outTmplP(w, ptmplOverboardPageErr, code, ctx)
		return
	}
	if !l.D.HasBackRefs {
		for i := range l.D.Threads {
			ib0.ProcessBackReferences(
				l.D.Threads[i].BoardName, &l.D.Threads[i].IBCommonThread)
		}
		l.D.HasBackRefs = true
	}
	tr.outTmplP(w, ptmplOverboardPage, 200, l)
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
		tr.outTmplP(w, ptmplThreadErr, code, ctx)
		return
	}
	if !l.D.HasBackRefs {
		ib0.ProcessBackReferences(l.D.Board.Name, &l.D.IBCommonThread)
		l.D.HasBackRefs = true
	}
	tr.outTmplP(w, ptmplThread, 200, l)
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
		tr.outTmplP(w, ptmplThreadCatalogErr, code, ctx)
		return
	}
	tr.outTmplP(w, ptmplThreadCatalog, 200, l)
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
		tr.outTmplR(w, rtmplCreatedBoard, 200, l)
	} else {
		tr.outTmplR(w, rtmplCreatedBoardErr, code, l)
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
			tr.outTmplR(w, rtmplCreatedThread, 200, l)
		} else {
			tr.outTmplR(w, rtmplCreatedThreadErr, code, l)
		}
	} else {
		if err == nil {
			tr.outTmplR(w, rtmplCreatedPost, 200, l)
		} else {
			tr.outTmplR(w, rtmplCreatedPostErr, code, l)
		}
	}
}
