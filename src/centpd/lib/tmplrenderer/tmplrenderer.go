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
	"centpd/lib/webcaptcha"
	ib0 "centpd/lib/webib0"

	"github.com/BurntSushi/toml"
)

var _ renderer.Renderer = (*TmplRenderer)(nil)

// page content
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
	rtmplCaptchaInclude

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
	"captcha_include",
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
	p    ib0.IBProvider
	tp   [ptmplMax]tmplThing
	tr   [rtmplMax]tmplThing
	m    msgFmtCfg
	l    Logger
	ni   NodeInfo
	wc   *webcaptcha.WebCaptcha
	scap bool // simple captcha
	ssi  bool
	esi  bool
}

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
		dir:       cfg.TemplateDir,
		env:       &tr.ni,
		staticdir: cfg.StaticDir,
	}
	if tr.wc != nil {
		if tr.scap {
			mc.captchamode = "simple"
		} else if tr.wc.UseCookies {
			mc.captchamode = "cookie"
		} else if tr.ssi {
			mc.captchamode = "ssi"
		} else if tr.esi {
			mc.captchamode = "esi"
		} else {
			panic("wtf")
		}
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
		base := ""
		if i&1 == 0 {
			base = "content"
		}
		t, ct, e := doFullTemplate(base, pnames[i])
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
	NodeInfo    NodeInfo
	WebCaptcha  *webcaptcha.WebCaptcha
	SSI         bool
	ESI         bool
	StaticDir   string
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
	w http.ResponseWriter, tt *tmplThing,
	tname string, code int, d interface{}) {

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

func (tr *TmplRenderer) newCaptchaKey(w http.ResponseWriter) string {
	if tr.scap {
		// make page containing key uncacheable
		w.Header().Set(
			"Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		return tr.wc.NewKey()
	} else {
		return ""
	}
}

func NewTmplRenderer(
	p ib0.IBProvider, cfg TmplRendererCfg) (
	tr *TmplRenderer, err error) {

	tr = &TmplRenderer{
		p:  p,
		ni: cfg.NodeInfo,
		wc: cfg.WebCaptcha,
		scap: cfg.WebCaptcha != nil &&
			!cfg.WebCaptcha.UseCookies &&
			!cfg.SSI && !cfg.ESI,
		ssi: cfg.SSI,
		esi: cfg.ESI,
	}

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
