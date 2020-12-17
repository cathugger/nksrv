package fileproxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type FileProxy struct {
	p *httputil.ReverseProxy
}

func NewFileProxy(upstream string) FileProxy {
	uri, err := url.ParseRequestURI(upstream)
	if err != nil {
		panic(err)
	}
	return FileProxy{
		p: httputil.NewSingleHostReverseProxy(uri),
	}
}

func (p FileProxy) ServeSrc(w http.ResponseWriter, r *http.Request, id string) {
	r.URL.Path = "/_src/" + id
	p.p.ServeHTTP(w, r)
	return
}

func (p FileProxy) ServeThm(w http.ResponseWriter, r *http.Request, id string) {
	r.URL.Path = "/_thm/" + id
	p.p.ServeHTTP(w, r)
	return
}
