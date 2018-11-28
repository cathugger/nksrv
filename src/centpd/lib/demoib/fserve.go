package demoib

import (
	"net/http"

	fsd "centpd/lib/fservedir"
	hfp "centpd/lib/httpibfileprovider"
	sp "centpd/lib/staticprovider"
)

var _ hfp.HTTPFileProvider = (*IBProviderDemo)(nil)
var _ sp.StaticProvider = (*IBProviderDemo)(nil)

var (
	srcServe    = fsd.NewFServeDir("_demo/demoib0/src")
	thmServe    = fsd.NewFServeDir("_demo/demoib0/thm")
	staticServe = fsd.NewFServeDir("_demo/demoib0/static")
)

func (IBProviderDemo) ServeSrc(w http.ResponseWriter, r *http.Request, id string) {
	srcServe.FServe(w, r, id)
}

func (IBProviderDemo) ServeThm(w http.ResponseWriter, r *http.Request, id string) {
	thmServe.FServe(w, r, id)
}

func (IBProviderDemo) ServeStatic(w http.ResponseWriter, r *http.Request, id string) {
	staticServe.FServe(w, r, id)
}
