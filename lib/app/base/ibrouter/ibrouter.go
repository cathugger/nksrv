package ibrouter

// simple html and webapi server

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"sync/atomic"
	"unsafe"

	"nksrv/lib/app/base/captchainfo"
	fsd "nksrv/lib/app/base/fservedir"
	fp "nksrv/lib/app/base/httpibfileprovider"
	wc "nksrv/lib/app/base/webcaptcha"
	"nksrv/lib/app/renderer"
	ib0 "nksrv/lib/app/webib0"
	"nksrv/lib/mail/form"
	"nksrv/lib/utils/handler"
	. "nksrv/lib/utils/logx"
)

type Cfg struct {
	Logger          *LoggerX
	HTMLRenderer    renderer.Renderer // handles everything else?
	StaticDir       *fsd.FServeDir
	FileProvider    fp.HTTPFileProvider   // handles _src and _thm
	APIHandler      http.Handler          // handles _api
	WebPostProvider ib0.IBWebPostProvider // handles html form submissions
	WebCaptcha      *wc.WebCaptcha
	CaptchaInfo     captchainfo.CaptchaInfo
	SSI             bool
	ESI             bool
	// fallback?
}

func localRedirect(w http.ResponseWriter, r *http.Request, newPath string) {
	if q := r.URL.RawQuery; q != "" {
		newPath += "?" + q
	}
	w.Header().Set("Location", newPath)
	w.WriteHeader(http.StatusMovedPermanently)
}

func handlePageNum(
	w http.ResponseWriter, r *http.Request, pn string) (ok bool, pni uint32) {

	if pn != "" {
		pnu, e := strconv.ParseUint(pn, 10, 32)
		if e != nil {
			// not found because invalid
			// TODO custom 404 pages
			// XXX maybe redirect to "./" there too?
			http.NotFound(w, r)
			return
		}
		if pnu <= 1 {
			// redirect to have uniform url
			// not http.Redirect because it uses full path and
			// we'd need reparse at this point as it's modified
			localRedirect(w, r, "./")
			return
		}
		// UI-visible stuff starts from 1, but internally from 0
		pni = uint32(pnu - 1)
	}
	ok = true
	return
}

type IBRouterCtl struct {
	p_HTMLRenderer unsafe.Pointer
}

func (c *IBRouterCtl) SetHTMLRenderer(r renderer.Renderer) {
	atomic.StorePointer(&c.p_HTMLRenderer, unsafe.Pointer(&r))
}
func (c *IBRouterCtl) GetHTMLRenderer() renderer.Renderer {
	return *(*renderer.Renderer)(atomic.LoadPointer(&c.p_HTMLRenderer))
}

var fileFieldsCheck = form.FieldsCheckFunc(ib0.IBWebFormFileFields)

func NewIBRouter(cfg Cfg) (http.Handler, *IBRouterCtl) {

	log := NewLogToX(*cfg.Logger, "ibrouter")

	c := new(IBRouterCtl)
	c.SetHTMLRenderer(cfg.HTMLRenderer)

	h_root := handler.NewCleanPath()

	h := handler.NewSimplePath()
	h_root.Handle(h)

	if cfg.APIHandler != nil {
		h.Handle("/_api", true, cfg.APIHandler)
	}

	if cfg.FileProvider != nil {
		h_src := handler.NewMethod().Handle("GET", handler.NewRegexPath().
			Handle("/{{id:[^_./][^/]*}}(?:/[^/]*)?", false,
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						id := r.Context().Value("id").(string)
						log.LogPrintf(DEBUG, "src %q", id)
						cfg.FileProvider.ServeSrc(w, r, id)
					})))
		h_thm := handler.NewMethod().Handle("GET", handler.NewRegexPath().
			Handle("/{{id:[^_./][^/]*}}", false, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					id := r.Context().Value("id").(string)
					log.LogPrintf(DEBUG, "thm %q", id)
					cfg.FileProvider.ServeThm(w, r, id)
				})))
		h.
			Handle("/_src", true, h_src).
			Handle("/_thm", true, h_thm)
	}

	if cfg.StaticDir != nil {
		h_static := handler.NewMethod().Handle("GET",
			handler.NewRegexPath().
				Handle("/{{id:[^_./][^/]*(?:/[^_./][^/]*)*}}?",
					false, http.HandlerFunc(func(
						w http.ResponseWriter, r *http.Request) {

						id := r.Context().Value("id").(string)
						log.LogPrintf(DEBUG, "static %q", id)
						cfg.StaticDir.FServe(w, r, id)
					})))
		h.Handle("/_static", true, h_static)
	}

	if cfg.HTMLRenderer != nil {
		h_html := handler.NewMethod()
		h.Fallback(h_html)

		h_get := handler.NewSimplePath()
		h_html.Handle("GET", h_get)

		h_get.Handle("/", false,
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					c.GetHTMLRenderer().ServeBoardList(w, r)
				}))

		h_get_overboard := handler.NewRegexPath()
		h_get.Handle("/_ukko", true, h_get_overboard)
		h_get.Handle("/*", true, h_get_overboard)
		h_get_overboard.Handle("/{{pn:[0-9]*}}", false,
			http.HandlerFunc(func(
				w http.ResponseWriter, r *http.Request) {

				pn := r.Context().Value("pn").(string)
				ok, pni := handlePageNum(w, r, pn)
				if !ok {
					return
				}

				log.LogPrintf(DEBUG, "overboard-page %d", pni)

				c.GetHTMLRenderer().
					ServeOverboardPage(w, r, pni)
			}))
		h_get_overboard.Handle("/catalog", false,
			http.HandlerFunc(func(
				w http.ResponseWriter, r *http.Request) {

				log.LogPrintf(DEBUG, "overboard-catalog")

				c.GetHTMLRenderer().
					ServeOverboardCatalog(w, r)
			}))

		h_getr := handler.NewRegexPath()
		h_get.Fallback(h_getr)

		h_getr_board := handler.NewRegexPath()
		h_getr.Handle("/{{b:[^_./][^/]*}}", true, h_getr_board)
		h_getr.Handle("/_{{b:[_.][^/]*}}", true, h_getr_board) // escaped
		// TODO handle boards not ending with slash

		h_getr_board.Handle("/{{pn:[0-9]*}}", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				pn := r.Context().Value("pn").(string)
				ok, pni := handlePageNum(w, r, pn)
				if !ok {
					return
				}
				log.LogPrintf(DEBUG, "board-page %q %d", b, pni)
				c.GetHTMLRenderer().ServeThreadListPage(w, r, b, pni)
			}))
		h_getr_board.Handle("/catalog", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				log.LogPrintf(DEBUG, "board-catalog %q", b)
				c.GetHTMLRenderer().ServeThreadCatalog(w, r, b)
			}))

		h_getr_board.Handle("/thread/{{t}}(?:/[^/]*)?", false,
			http.HandlerFunc(func(
				w http.ResponseWriter, r *http.Request) {

				b := r.Context().Value("b").(string)
				t := r.Context().Value("t").(string)
				log.LogPrintf(DEBUG, "board-thread %q %q", b, t)
				c.GetHTMLRenderer().ServeThread(w, r, b, t)
			}))
	}

	fparam, fopener, tfields := cfg.WebPostProvider.IBGetPostParams()
	textFieldsCheck := func(field string) bool {
		return tfields(field) || field == "board" || field == "thread"
	}

	// TODO maybe should do it in more REST-ful way and add to html handler?
	if cfg.WebPostProvider != nil {
		if cfg.HTMLRenderer == nil {
			panic("WebPostProvider requires HTMLRenderer")
		}

		h_post := handler.NewMethod()
		h_post.Handle("POST", http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ct, param, e :=
					mime.ParseMediaType(r.Header.Get("Content-Type"))
				if e != nil {
					http.Error(w,
						fmt.Sprintf("failed to parse content type: %v", e),
						http.StatusBadRequest)
					return
				}
				if ct != "multipart/form-data" || param["boundary"] == "" {
					http.Error(w, "bad Content-Type", http.StatusBadRequest)
					return
				}

				var err error
				f, err := fparam.ParseForm(
					r.Body, param["boundary"],
					textFieldsCheck, fileFieldsCheck, fopener)
				if err != nil {
					// TODO
					http.Error(w,
						fmt.Sprintf("error parsing form: %v", err),
						http.StatusBadRequest)
					return
				}
				if len(f.Values["board"]) != 1 ||
					len(f.Values["thread"]) > 1 {

					http.Error(w,
						"invalid form params",
						http.StatusBadRequest)
					return
				}
				board := f.Values["board"][0]
				var rInfo ib0.IBPostedInfo
				var code int
				if len(f.Values["thread"]) == 0 ||
					f.Values["thread"][0] == "" {

					log.LogPrintf(DEBUG, "post-thread b:%q", board)

					rInfo, err =
						cfg.WebPostProvider.IBPostNewThread(w, r, f, board)
					if err != nil {
						err, code = ib0.UnpackWebPostError(err)
					}

					c.GetHTMLRenderer().DressPostResult(
						w, rInfo, true, err, code)
				} else {
					thread := f.Values["thread"][0]

					log.LogPrintf(DEBUG, "post-reply b:%q t:%q", board, thread)

					rInfo, err = cfg.WebPostProvider.
						IBPostNewReply(w, r, f, board, thread)
					if err != nil {
						err, code = ib0.UnpackWebPostError(err)
					}

					c.GetHTMLRenderer().DressPostResult(
						w, rInfo, false, err, code)
				}
			}))
		h.Handle("/_post/post", false, h_post)
	}

	if cfg.WebCaptcha != nil {
		h_captchaget := handler.NewSimplePath().
			Handle("/captcha.png", false, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					// since we're only accepting GET, don't need to check type
					err := r.ParseForm()
					if err != nil {
						http.Error(w, "bad query", http.StatusBadRequest)
						return
					}

					key := r.Form.Get("key")
					err, code := cfg.WebCaptcha.ServeCaptchaPNG(
						w, r, key,
						cfg.CaptchaInfo.Width, cfg.CaptchaInfo.Height)
					if err != nil {
						http.Error(w, err.Error(), code)
					}
				}))
		if !cfg.WebCaptcha.UseCookies && (cfg.SSI || cfg.ESI) {
			h_captchaget.Handle("/include", false, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					c.GetHTMLRenderer().WebCaptchaInclude(w, r)
				}))
		}
		h_captcha := handler.NewMethod().Handle("GET", h_captchaget)
		if !cfg.WebCaptcha.UseCookies {
			h.Handle("/_captcha", true, h_captcha)
		} else {
			h.Handle("/_post", true, h_captcha)
		}
	}

	return h_root, c
}
