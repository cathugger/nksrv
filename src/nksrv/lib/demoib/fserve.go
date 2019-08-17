package demoib

import (
	"net/http"

	fsd "nksrv/lib/fservedir"
	hfp "nksrv/lib/httpibfileprovider"
	sp "nksrv/lib/staticprovider"
)

var _ hfp.HTTPFileProvider = (*IBProviderDemo)(nil)
var _ sp.StaticProvider = (*IBProviderDemo)(nil)

/*
 * infinite caching, templating engine will provide proper ?v= params
 */
var srccfg = fsd.Config{
	CacheControl: "public, max-age=31536000, no-transform, immutable"}
var thmcfg = fsd.Config{CacheControl: "public, max-age=31536000, immutable"}
var staticcfg = fsd.Config{CacheControl: "public, max-age=31536000"}

var (
	srcDir    = fsd.NewFServeDir("_demo/demoib0/src", srccfg)
	thmDir    = fsd.NewFServeDir("_demo/demoib0/thm", thmcfg)
	StaticDir = fsd.NewFServeDir("_demo/demoib0/static", staticcfg)
)

func (IBProviderDemo) ServeSrc(
	w http.ResponseWriter, r *http.Request, id string) {

	srcDir.FServe(w, r, id)
}

func (IBProviderDemo) ServeThm(
	w http.ResponseWriter, r *http.Request, id string) {

	thmDir.FServe(w, r, id)
}

func (IBProviderDemo) ServeStatic(
	w http.ResponseWriter, r *http.Request, id string) {

	StaticDir.FServe(w, r, id)
}
