package apirouter

// simple html and webapi server

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"

	"nekochan/lib/handler"
	"nekochan/lib/renderer"
	ib0 "nekochan/lib/webib0"
)

type Cfg struct {
	Renderer        renderer.Renderer     // handles everything else?
	WebPostProvider ib0.IBWebPostProvider // handles html form submissions
	// fallback?
}

func NewAPIRouter(cfg Cfg) http.Handler {
	h_root := handler.NewCleanPath()

	h := handler.NewSimplePath()
	h_root.Handle(h)

	if cfg.Renderer == nil {
		panic("nil renderer not allowed")
	}

	h_bcontent := handler.NewRegexPath()
	h_bcontent.Handle("/pages/{{n:[0-9]+}}", false,
		handler.NewMethod().Handle("GET", http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				sn := r.Context().Value("n").(string)
				n, e := strconv.ParseUint(sn, 10, 32)
				if e != nil {
					// not found because invalid
					// TODO custom 404 pages
					http.NotFound(w, r)
					return
				}
				cfg.Renderer.ServeThreadListPage(w, r, b, uint32(n))
			})))
	h_bcontent.Handle("/catalog", false,
		handler.NewMethod().Handle("GET", http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				cfg.Renderer.ServeThreadCatalog(w, r, b)
			})))
	h_threads := handler.NewMethod().Handle("GET", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			b := r.Context().Value("b").(string)
			t := r.Context().Value("t").(string)
			cfg.Renderer.ServeThread(w, r, b, t)
		}))
	if cfg.WebPostProvider != nil {
		h_threads.Handle("POST", http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b := r.Context().Value("b").(string)
				t := r.Context().Value("t").(string)

				ct, param, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if e != nil {
					http.Error(w, fmt.Sprintf("failed to parse content type: %v", e), http.StatusBadRequest)
					return
				}
				if ct != "multipart/form-data" || param["boundary"] == "" {
					http.Error(w, "bad Content-Type", http.StatusBadRequest)
					return
				}

				fparam, fopener := cfg.WebPostProvider.IBGetPostParams()
				textFields := []string{
					ib0.IBWebFormTextTitle,
					ib0.IBWebFormTextMessage,
				}
				var err error
				f, err := fparam.ParseForm(r.Body, param["boundary"], textFields, ib0.IBWebFormFileFields, fopener)
				if err != nil {
					// TODO
					http.Error(w, fmt.Sprintf("error parsing form: %v", err), http.StatusBadRequest)
					return
				}

				var rInfo ib0.IBPostedInfo
				var code int
				rInfo, err, code = cfg.WebPostProvider.
					IBPostNewReply(r, f, b, t)

				cfg.Renderer.DressPostResult(w, rInfo, err, code)
			}))
	}
	h_bcontent.Handle("/threads/{{t}}", false, h_threads)
	if cfg.WebPostProvider != nil {
		h_bcontent.Handle("/", false,
			handler.NewMethod().Handle("POST", http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					b := r.Context().Value("b").(string)

					ct, param, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
					if e != nil {
						http.Error(w, fmt.Sprintf("failed to parse content type: %v", e), http.StatusBadRequest)
						return
					}
					if ct != "multipart/form-data" || param["boundary"] == "" {
						http.Error(w, "bad Content-Type", http.StatusBadRequest)
						return
					}

					fparam, fopener := cfg.WebPostProvider.IBGetPostParams()
					textFields := []string{
						ib0.IBWebFormTextTitle,
						ib0.IBWebFormTextMessage,
					}
					var err error
					f, err := fparam.ParseForm(r.Body, param["boundary"], textFields, ib0.IBWebFormFileFields, fopener)
					if err != nil {
						// TODO
						http.Error(w, fmt.Sprintf("error parsing form: %v", err), http.StatusBadRequest)
						return
					}

					var rInfo ib0.IBPostedInfo
					var code int
					rInfo, err, code = cfg.WebPostProvider.
						IBPostNewThread(r, f, b)

					cfg.Renderer.DressPostResult(w, rInfo, err, code)
				})))
	}

	h_boards := handler.NewRegexPath()
	h_boards.Handle("/", false,
		handler.NewMethod().Handle("GET",
			http.HandlerFunc(cfg.Renderer.ServeBoardList)))
	h_boards.Handle("/{{b}}", true, h_bcontent)

	h.Handle("/boards", true, h_boards)
	h.Fallback(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))

	return h_root
}
