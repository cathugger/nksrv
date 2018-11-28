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
	recognised  bool
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
	var tt *tmplTOML
	var t *template.Template
	var f []byte
	var err error

	tn := "templates.toml"
	f, err = ioutil.ReadFile(path.Join(cfg.TemplateDir, tn))
	if err != nil {
		tr.l.LogPrintf(INFO, "couldn't read %q: %v", tn, err)
	} else {
		fukugo := make(tmplTOML)
		tt = &fukugo
		err = toml.Unmarshal(f, tt)
		if err != nil {
			tr.l.LogPrintf(ERROR, "failed to parse TOML file %q: %v", tn, err)
			return fmt.Errorf("failed to parse TOML file %q: %v", tn, err)
		}
	}
	var dcharset string
	if tt != nil {
		s, ok := (*tt)["default"]
		if ok {
			(*tt)["default"].recognised = true
			dcharset = s.Charset
		}
	}
	if dcharset == "" {
		dcharset = "utf-8"
	}
	for i := 0; i < tmplMax; i++ {
		filename := names[i] + ".tmpl"
		charset := dcharset
		// default content type for error pages is text/plain
		var contenttype string
		if i&1 == 0 {
			contenttype = "text/html"
		} else {
			contenttype = "text/plain"
		}
		if tt != nil {
			s, ok := (*tt)[names[i]]
			if ok {
				(*tt)[names[i]].recognised = true
				if s.FileName != "" {
					filename = s.FileName
				}
				if s.ContentType != "" {
					contenttype = s.ContentType
				}
				if s.Charset != "" {
					charset = s.Charset
				}
			}
		}
		lenc := strings.ToLower(charset)
		var cset string
		switch lenc {
		case "utf-8", "utf8":
			tr.t[i].w = nopWCCreator
			cset = charset[:3] + "-8"
		case "ascii", "us-ascii", "iso-8859-1":
			tr.t[i].w = nopWCCreator
			cset = charset
		case "utf-16", "utf16", "utf-16/be", "utf16/be":
			tr.t[i].w = utf16beWCCreator
			cset = charset[:3] + "-16"
		case "utf-16/le", "utf16/le":
			tr.t[i].w = utf16leWCCreator
			cset = charset[:3] + "-16"
		case "utf16be", "utf-16be":
			tr.t[i].w = utf16beNBOMWCCreator
			cset = charset[:3] + "-16" + charset[len(charset)-2:]
		case "utf16le", "utf-16le":
			tr.t[i].w = utf16leNBOMWCCreator
			cset = charset[:3] + "-16" + charset[len(charset)-2:]
		default:
			tr.l.LogPrintf(ERROR, "unknown charset: %s", charset)
			return fmt.Errorf("unknown charset: %s", charset)
		}
		mt, par, err := mime.ParseMediaType(contenttype)
		if err != nil {
			tr.l.LogPrintf(ERROR, "couldn't parse Content-Type %q: %v",
				contenttype, err)
			return fmt.Errorf("couldn't parse Content-Type %q: %v",
				contenttype, err)
		}

		par["charset"] = cset
		tr.t[i].m = mime.FormatMediaType(mt, par)

		f, err = ioutil.ReadFile(path.Join(cfg.TemplateDir, filename))
		if err != nil {
			tr.l.LogPrintf(ERROR, "failed to read %q: %v", filename, err)
			return fmt.Errorf("failed to read %q: %v", filename, err)
		}
		t = template.New(filename).Funcs(funcs)
		t, err = t.Parse(string(f))
		if err != nil {
			tr.l.LogPrintf(ERROR, "failed to parse template file %q: %v",
				filename, err)
			return fmt.Errorf("failed to parse template file %q: %v",
				filename, err)
		}
		tr.t[i].t = t
	}
	if tt != nil {
		for n := range *tt {
			if !(*tt)[n].recognised {
				tr.l.LogPrintf(WARN, "unrecognised %q section %q", tn, n)
			}
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

	tn := "message.toml"
	f, err = ioutil.ReadFile(path.Join(cfg.TemplateDir, tn))
	if err != nil {
		tr.l.LogPrintf(ERROR, "failed to read %q: %v", tn, err)
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
