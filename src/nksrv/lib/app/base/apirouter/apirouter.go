package apirouter

// simple html and webapi server

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strconv"

	"nksrv/lib/app/base/oauth2"
	"nksrv/lib/app/renderer"
	ib0 "nksrv/lib/app/webib0"
	"nksrv/lib/mail/form"
	"nksrv/lib/utils/handler"
)

type Cfg struct {
	Renderer        renderer.Renderer     // handles everything else?
	WebPostProvider ib0.IBWebPostProvider // handles html form submissions
	Auth            *oauth2.IBOAuth2
	// fallback?
}

var formFileFieldFunc = form.FieldsCheckFunc(ib0.IBWebFormFileFields)

func eatMessagePost(
	w http.ResponseWriter, r *http.Request,
	wpp ib0.IBWebPostProvider) (f form.Form, ok bool) {

	ct, param, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil {
		http.Error(
			w, fmt.Sprintf("failed to parse content type: %v", e),
			http.StatusBadRequest)
		return
	}
	if ct != "multipart/form-data" || param["boundary"] == "" {
		http.Error(w, "bad Content-Type", http.StatusBadRequest)
		return
	}

	fparam, fopener, tfields := wpp.IBGetPostParams()
	var err error
	f, err = fparam.ParseForm(
		r.Body, param["boundary"],
		tfields, formFileFieldFunc,
		fopener)
	if err != nil {
		// TODO
		http.Error(
			w, fmt.Sprintf("error parsing form: %v", err),
			http.StatusBadRequest)
		return
	}

	ok = true
	return
}

func httpErrorBadRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "bad request", http.StatusBadRequest)
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
					httpErrorBadRequest(w, r)
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

				f, ok := eatMessagePost(w, r, cfg.WebPostProvider)
				if !ok {
					return
				}

				var code int
				rInfo, err := cfg.WebPostProvider.
					IBPostNewReply(w, r, f, b, t)
				if err != nil {
					err, code = ib0.UnpackWebPostError(err)
				}

				cfg.Renderer.DressPostResult(w, rInfo, false, err, code)
			}))
	}
	h_bcontent.Handle("/threads/{{t}}", false, h_threads)

	if cfg.WebPostProvider != nil {
		h_bcontent.Handle("/", false,
			handler.NewMethod().Handle("POST", http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					b := r.Context().Value("b").(string)

					f, ok := eatMessagePost(w, r, cfg.WebPostProvider)
					if !ok {
						return
					}

					var code int
					rInfo, err := cfg.WebPostProvider.
						IBPostNewThread(w, r, f, b)
					if err != nil {
						err, code = ib0.UnpackWebPostError(err)
					}

					cfg.Renderer.DressPostResult(w, rInfo, true, err, code)
				})))
	}

	h_boards := handler.NewRegexPath()

	h_boardsroot := handler.NewMethod().
		Handle("GET", http.HandlerFunc(cfg.Renderer.ServeBoardList))

	if cfg.WebPostProvider != nil {

		h_boardsroot.Handle("POST", http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				ct, _, e :=
					mime.ParseMediaType(r.Header.Get("Content-Type"))
				if e != nil {
					http.Error(
						w, fmt.Sprintf("failed to parse content type: %v", e),
						http.StatusBadRequest)
					return
				}
				if ct != "application/json" {
					http.Error(w, "bad Content-Type", http.StatusBadRequest)
					return
				}

				jd := json.NewDecoder(r.Body)
				nbi := cfg.WebPostProvider.IBDefaultBoardInfo()
				e = jd.Decode(&nbi)
				if e != nil {
					http.Error(
						w, fmt.Sprintf("failed to parse content: %v", e),
						http.StatusBadRequest)
					return
				}

				var code int
				e = cfg.WebPostProvider.IBPostNewBoard(w, r, nbi)
				if e != nil {
					e, code = ib0.UnpackWebPostError(e)
				}
				cfg.Renderer.DressNewBoardResult(w, nbi.Name, e, code)

			}))

	}
	h_boards.Handle("/", false, h_boardsroot)
	h_boards.Handle("/{{b}}", true, h_bcontent)

	if cfg.WebPostProvider != nil {

		h_bthing := handler.NewMethod().
			Handle("PATCH", http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {

					ct, _, e :=
						mime.ParseMediaType(r.Header.Get("Content-Type"))
					if e != nil {
						http.Error(w,
							fmt.Sprintf("failed to parse content type: %v", e),
							http.StatusBadRequest)
						return
					}
					if ct != "application/json" {
						http.Error(
							w, "bad Content-Type", http.StatusBadRequest)
						return
					}

					jd := json.NewDecoder(r.Body)
					nbi := cfg.WebPostProvider.IBDefaultBoardInfo()
					e = jd.Decode(&nbi)
					if e != nil {
						http.Error(
							w, fmt.Sprintf("failed to parse content: %v", e),
							http.StatusBadRequest)
						return
					}

					nbi.Name = r.Context().Value("b").(string)

					var code int
					e = cfg.WebPostProvider.IBUpdateBoard(w, r, nbi)
					if e != nil {
						e, code = ib0.UnpackWebPostError(e)
					}
					cfg.Renderer.DressNewBoardResult(w, nbi.Name, e, code)

				})).
			Handle("DELETE", http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					b := r.Context().Value("b").(string)

					var code int
					e := cfg.WebPostProvider.IBDeleteBoard(w, r, b)
					if e != nil {
						e, code = ib0.UnpackWebPostError(e)
						http.Error(w, e.Error(), code)
						return
					}

					http.Error(w, "deleted", 200)
				}))

		h_boards.Handle("/{{b}}", false, h_bthing)

	}

	h.Handle("/boards", true, h_boards)

	h_overboard := handler.NewRegexPath()

	h_overboard.Handle("/pages/{{n:[0-9]+}}", false, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			sn := r.Context().Value("n").(string)
			n, e := strconv.ParseUint(sn, 10, 32)
			if e != nil {
				httpErrorBadRequest(w, r)
				return
			}
			cfg.Renderer.ServeOverboardPage(w, r, uint32(n))
		}))
	h_overboard.Handle("/catalog", false, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			cfg.Renderer.ServeOverboardCatalog(w, r)
		}))

	h.Handle("/overboard", true,
		handler.NewMethod().Handle("GET", h_overboard))

	if cfg.Auth != nil {
		h.Handle("/auth/login", false, http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				ct, _, e :=
					mime.ParseMediaType(r.Header.Get("Content-Type"))
				if e != nil {
					http.Error(
						w, fmt.Sprintf("failed to parse content type: %v", e),
						http.StatusBadRequest)
					return
				}
				if ct != "application/json" {
					http.Error(w, "bad Content-Type", http.StatusBadRequest)
					return
				}

				jd := json.NewDecoder(r.Body)
				logininfo := struct {
					User string `json:"user"`
					Pass string `json:"pass"`
				}{}
				e = jd.Decode(&logininfo)
				if e != nil {
					http.Error(
						w, fmt.Sprintf("failed to parse content: %v", e),
						http.StatusBadRequest)
					return
				}

				tok, err, code := cfg.Auth.Login(r, logininfo.User, logininfo.Pass)
				if err != nil {
					http.Error(w, err.Error(), code)
					return
				}
				w.Write([]byte(tok))
			}))
	}

	return h_root
}
