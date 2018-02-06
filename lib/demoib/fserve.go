package demoib

import (
	fsd "../fservedir"
	hfp "../httpibfileprovider"
	"net/http"
)

var _ hfp.HTTPFileProvider = (*IBProviderDemo)(nil)

var (
	srcServe = fsd.NewFServeDir("_demo/demoib0/src")
	thmServe = fsd.NewFServeDir("_demo/demoib0/thm")
)

func (IBProviderDemo) ServeSrc(w http.ResponseWriter, r *http.Request, id string) {
	srcServe.FServe(w, r, id)
}

func (IBProviderDemo) ServeThm(w http.ResponseWriter, r *http.Request, id string) {
	thmServe.FServe(w, r, id)
}
