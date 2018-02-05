package demoib

import (
	fsd "../fservedir"
	"../httpibfileprovider"
	"net/http"
)

var _ webib0.HTTPFileProvider = (*IBProviderDemo)(nil)

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
