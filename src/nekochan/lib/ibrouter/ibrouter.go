package ibrouter

// simple html and webapi server

import (
	"net/http"
	"net/url"
	"strconv"

	"nekochan/lib/handler"
	fp "nekochan/lib/httpibfileprovider"
	"nekochan/lib/renderer"
	sp "nekochan/lib/staticprovider"
)

type Cfg struct {
	HTMLRenderer   renderer.Renderer // handles everything else?
	StaticProvider sp.StaticProvider
	FileProvider   fp.HTTPFileProvider // handles _src and _thm
	APIHandler     http.Handler        // handles _api
	// fallback?
	// http posting?
}

func NewIBRouter(cfg Cfg) http.Handler {
	h_root := handler.NewCleanPath()

	h := handler.NewSimplePath()
	h_root.Handle(h)

	if cfg.APIHandler != nil {
		h.Handle("/_api", true, cfg.APIHandler)
	}

	if cfg.FileProvider != nil {
		h_src := handler.NewMethod().Handle("GET", handler.NewRegexPath().
			Handle("/{{id:[^_./][^/]*}}(?:/[^/]*)?", false, http.HandlerFunc(
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

	if cfg.StaticProvider != nil {
		h_static := handler.NewMethod().Handle("GET", handler.NewRegexPath().
			Handle("/{{id:[^_./][^/]*}}(?:/[^/]*)?", false, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					id := r.Context().Value("id").(string)
					cfg.StaticProvider.ServeStatic(w, r, id)
				})))
		h.Handle("/_static", true, h_static)
	}

	if cfg.HTMLRenderer != nil {
		h_html := handler.NewMethod()
		h.Fallback(h_html)

		h_get := handler.NewSimplePath()
		h_html.Handle("GET", h_get)

		h_get.Handle("/", false, http.HandlerFunc(cfg.HTMLRenderer.ServeBoardList))

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
				pni := uint32(0)
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
						// need to re-parse URL, because at this point it's modified
						ru, _ := url.ParseRequestURI(r.RequestURI)
						r.URL = ru
						http.Redirect(w, r, "./", http.StatusTemporaryRedirect)
						return
					}
					// UI-visible stuff starts from 1, but internally from 0
					pni = uint32(pnu - 1)
				}
				cfg.HTMLRenderer.ServeThreadListPage(w, r, b, pni)
			}))
		h_getbr.Handle("/catalog", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				cfg.HTMLRenderer.ServeThreadCatalog(w, r, b)
			}))

		h_getbr.Handle("/thread/{{t}}(?:/[^/]*)?", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				t := r.Context().Value("t").(string)
				cfg.HTMLRenderer.ServeThread(w, r, b, t)
			}))
	}

	return h_root
}
