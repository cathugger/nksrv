package handler

import (
	"net/http"
)

type simplePathRoute struct {
	handler http.Handler
	path    string
	strip   bool
}

type SimplePath struct {
	routes   []simplePathRoute
	fallback http.Handler
}

func NewSimplePath() *SimplePath {
	return new(SimplePath).Initialize()
}

func (p *SimplePath) Initialize() *SimplePath {
	p.fallback = http.HandlerFunc(badRequest)
	return p
}

func (p *SimplePath) Handle(path string, strip bool, handler http.Handler) *SimplePath {
	p.routes = append(p.routes, simplePathRoute{
		handler: handler,
		path:    path,
		strip:   strip,
	})
	return p
}

func (p *SimplePath) Fallback(handler http.Handler) *SimplePath {
	p.fallback = handler
	return p
}

func (p *SimplePath) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for i := range p.routes {
		if !p.routes[i].strip {
			// exact match
			if r.URL.Path == p.routes[i].path {
				p.routes[i].handler.ServeHTTP(w, r)
				return
			}
		} else {
			// prefix match
			l := len(p.routes[i].path)
			if len(r.URL.Path) > l && r.URL.Path[:l] == p.routes[i].path && r.URL.Path[l] == '/' {
				// this is destructive
				r.URL.Path = r.URL.Path[l:]
				p.routes[i].handler.ServeHTTP(w, r)
				return
			}
		}
	}
	p.fallback.ServeHTTP(w, r)
}

var _ http.Handler = (*SimplePath)(nil)
