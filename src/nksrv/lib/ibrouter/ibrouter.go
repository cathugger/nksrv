package ibrouter

// simple html and webapi server

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"sync/atomic"
	"unsafe"

	"nksrv/lib/captchainfo"
	fsd "nksrv/lib/fservedir"
	"nksrv/lib/handler"
	fp "nksrv/lib/httpibfileprovider"
	"nksrv/lib/mail/form"
	"nksrv/lib/renderer"
	wc "nksrv/lib/webcaptcha"
	ib0 "nksrv/lib/webib0"
)

type Cfg struct {
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
						cfg.FileProvider.ServeSrc(w, r, id)
					})))
		h_thm := handler.NewMethod().Handle("GET", handler.NewRegexPath().
			Handle("/{{id:[^_./][^/]*}}", false, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					id := r.Context().Value("id").(string)
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

		h_get.Handle("/_ukko", true,
			handler.NewRegexPath().
				Handle("/{{pn:[0-9]*}}", false,
					http.HandlerFunc(func(
						w http.ResponseWriter, r *http.Request) {

						pn := r.Context().Value("pn").(string)
						ok, pni := handlePageNum(w, r, pn)
						if !ok {
							return
						}

						c.GetHTMLRenderer().
							ServeOverboardPage(w, r, pni)
					})).
				Handle("/catalog", false,
					http.HandlerFunc(func(
						w http.ResponseWriter, r *http.Request) {

						c.GetHTMLRenderer().
							ServeOverboardCatalog(w, r)
					})))

		h_getr := handler.NewRegexPath()
		h_get.Fallback(h_getr)

		h_getbr := handler.NewRegexPath()
		h_getr.Handle("/{{b:[^_./][^/]*}}", true, h_getbr)
		h_getr.Handle("/_{{b:[_.][^/]*}}", true, h_getbr)
		// TODO handle boards not ending with slash

		h_getbr.Handle("/{{pn:[0-9]*}}", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				pn := r.Context().Value("pn").(string)
				ok, pni := handlePageNum(w, r, pn)
				if !ok {
					return
				}
				c.GetHTMLRenderer().ServeThreadListPage(w, r, b, pni)
			}))
		h_getbr.Handle("/catalog", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				c.GetHTMLRenderer().ServeThreadCatalog(w, r, b)
			}))

		h_getbr.Handle("/thread/{{t}}(?:/[^/]*)?", false,
			http.HandlerFunc(func(
				w http.ResponseWriter, r *http.Request) {

				b := r.Context().Value("b").(string)
				t := r.Context().Value("t").(string)
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

					rInfo, err, code =
						cfg.WebPostProvider.IBPostNewThread(w, r, f, board)
					c.GetHTMLRenderer().DressPostResult(
						w, rInfo, true, err, code)
				} else {
					rInfo, err, code = cfg.WebPostProvider.
						IBPostNewReply(w, r, f, board, f.Values["thread"][0])
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
