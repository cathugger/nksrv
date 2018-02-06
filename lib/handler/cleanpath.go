package handler

import (
	"net/http"
	"path"
)

type CleanPath struct {
	handler http.Handler
}

// Return the canonical path for p, eliminating . and .. elements.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

func NewCleanPath() CleanPath {
	return CleanPath{}.Initialize()
}

func (cp CleanPath) Initialize() CleanPath {
	return cp
}

func (cp *CleanPath) Handle(handler http.Handler) *CleanPath {
	cp.handler = handler
	return cp
}

func (cp CleanPath) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "CONNECT" {
		// expects that r.URL.Path isn't modified
		np := cleanPath(r.URL.Path)
		if np != r.URL.Path {
			url := *r.URL
			url.Path = np
			http.Redirect(w, r, url.String(), http.StatusTemporaryRedirect)
			return
		}
	}
	cp.handler.ServeHTTP(w, r)
}

var _ http.Handler = (*CleanPath)(nil)
