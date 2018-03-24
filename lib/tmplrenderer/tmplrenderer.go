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
	"mime"
	"net/http"
	"path"
	"strings"
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

var names = [tmplMax]string{
	"board_list",
	"board_list_err",
	"thread_list_page",
	"thread_list_page_err",
	"thread_catalog",
	"thread_catalog_err",
	"thread",
	"thread_err",
}

type msgFmtTOML struct {
	PreMsg        string `toml:"pre_msg"`
	PostMsg       string `toml:"post_msg"`
	PreLine       string `toml:"pre_line"`
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
	p webib0.IBProvider
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
			tr.l.LogPrintf(ERROR, "couldn't parse Content-Type %q: %v", contenttype, err)
			return fmt.Errorf("couldn't parse Content-Type %q: %v", contenttype, err)
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
			tr.l.LogPrintf(ERROR, "failed to parse template file %q: %v", filename, err)
			return fmt.Errorf("failed to parse template file %q: %v", filename, err)
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

func tmplWC(w http.ResponseWriter, tr *TmplRenderer, num int, code int) io.WriteCloser {
	tt := &tr.t[num]
	w.Header().Set("Content-Type", tt.m)
	w.WriteHeader(code)
	return tt.w(w)
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

func (tr *TmplRenderer) execTmpl(t int, w io.WriteCloser, d interface{}) {
	err := tr.t[t].t.Execute(w, d)
	if err != nil {
		tr.l.LogPrintf(ERROR, "%s execution failed: %v", names[t], err)
	}
	w.Close()
}

func outTmpl(w http.ResponseWriter, tr *TmplRenderer, num int, code int, d interface{}) {
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
		PreMsg:       []byte(mtoml.PreMsg),
		PostMsg:      []byte(mtoml.PostMsg),
		FirstPreLine: []byte(mtoml.FirstPreLine),
		NextPreLine:  []byte(mtoml.NextPreLine),
		PostLine:     []byte(mtoml.PostLine),
		Newline:      []byte(mtoml.Newline),
	}
	if mtoml.FirstPreLine == "" {
		tr.m.FirstPreLine = []byte(mtoml.PreLine)
	}
	if mtoml.NextPreLine == "" {
		tr.m.NextPreLine = []byte(mtoml.PreLine)
	}

	t = template.New("pre_reference").Funcs(funcs)
	tr.m.preRefTmpl, err = t.Parse(mtoml.PreReference)
	if err != nil {
		return fmt.Errorf("failed to parse template %q: %v", mtoml.PreReference, err)
	}

	t = template.New("post_reference").Funcs(funcs)
	tr.m.postRefTmpl, err = t.Parse(mtoml.PostReference)
	if err != nil {
		return fmt.Errorf("failed to parse template %q: %v", mtoml.PostReference, err)
	}
	return nil
}

func NewTmplRenderer(p webib0.IBProvider, cfg TmplRendererCfg) (*TmplRenderer, error) {
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

func (tr *TmplRenderer) ServeBoardList(w http.ResponseWriter, r *http.Request) {
	l := &struct {
		D webib0.IBBoardList
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

func (tr *TmplRenderer) ServeThreadListPage(w http.ResponseWriter, r *http.Request, board string, page uint32) {
	l := &struct {
		D webib0.IBThreadListPage
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
			webib0.ProcessBackReferences(&l.D.Threads[i].IBCommonThread)
		}
		l.D.HasBackRefs = true
	}
	outTmpl(w, tr, tmplThreadListPage, 200, l)
}

func (tr *TmplRenderer) ServeThread(w http.ResponseWriter, r *http.Request, board, thread string) {
	l := &struct {
		D webib0.IBThreadPage
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
		webib0.ProcessBackReferences(&l.D.IBCommonThread)
		l.D.HasBackRefs = true
	}
	outTmpl(w, tr, tmplThread, 200, l)
}

func (tr *TmplRenderer) ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string) {
	l := &struct {
		D webib0.IBThreadCatalog
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
