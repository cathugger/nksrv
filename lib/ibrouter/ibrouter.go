package ibrouter

// simple html and webapi server

import (
	"../handler"
	fp "../httpibfileprovider"
	"../renderer"
	"net/http"
	"strconv"
)

type Cfg struct {
	HTMLRenderer renderer.Renderer   // handles everything else?
	FileProvider fp.HTTPFileProvider // handles _src and _thm
	APIHandler   http.Handler        // handles _api
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

	if cfg.HTMLRenderer != nil {
		h_html := handler.NewMethod()
		h.Fallback(h_html)

		h_get := handler.NewSimplePath()
		h_html.Handle("GET", h_get)

		h_get.Handle("/", false, http.HandlerFunc(cfg.HTMLRenderer.ServeBoardList))

		h_getr := handler.NewRegexPath()
		h_get.Fallback(h_getr)

		h_getb := handler.NewSimplePath()
		h_getr.Handle("/{{b:[^_./][^/]*}}", true, h_getb)
		h_getr.Handle("/_{{b:[_.][^/]*}}", true, h_getb)
		// TODO handle boards not ending with slash

		h_getb.Handle("/{{pn:[0-9]*}}", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				pn := r.Context().Value("pn").(string)
				pni := uint32(0)
				if pnu, e := strconv.ParseUint(pn, 10, 32); e == nil && pnu > 1 {
					// UI-visible stuff starts from 1, but internally from 0
					pni = uint32(pnu - 1)
				}
				cfg.HTMLRenderer.ServeThreadListPage(w, r, b, pni)
			}))
		h_getb.Handle("/catalog", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				cfg.HTMLRenderer.ServeThreadCatalog(w, r, b)
			}))

		h_getbr := handler.NewRegexPath()
		h_getb.Fallback(h_getbr)

		h_getbr.Handle("/thread/{{t}}(?:/[^/]*)?", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				t := r.Context().Value("t").(string)
				cfg.HTMLRenderer.ServeThread(w, r, b, t)
			}))
	}

	return h_root
}
