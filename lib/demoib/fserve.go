package demoib

import (
	"net/http"
	fsd "../fservedir"
	"../webib0"
)

type HTTPFileProvider interface {
	ServeSrc(w http.ResponseWriter, r *http.Request, id string)
	ServeThm(w http.ResponseWriter, r *http.Request, id string)
}

var _ webib0.HTTPFileProvider = (*IBProviderDemo)(nil)

var (
	srcServe = fsd.NewFServeDir("_demo/0/src")
	thmServe = fsd.NewFServeDir("_demo/0/thm")
)

func (IBProviderDemo) ServeSrc(w http.ResponseWriter, r *http.Request, id string) {
	srcServe.FServe(w, r, id)
}

func (IBProviderDemo) ServeThm(w http.ResponseWriter, r *http.Request, id string) {
	thmServe.FServe(w, r, id)
}
