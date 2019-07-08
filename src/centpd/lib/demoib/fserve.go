package demoib

import (
	"net/http"

	fsd "centpd/lib/fservedir"
	hfp "centpd/lib/httpibfileprovider"
	sp "centpd/lib/staticprovider"
)

var _ hfp.HTTPFileProvider = (*IBProviderDemo)(nil)
var _ sp.StaticProvider = (*IBProviderDemo)(nil)

/*
 * TODO:
 * static should ALSO have infinite caching BUT we should do something like this:
 * /_static/style.css?timestamp
 * this however will need templating engine awareness
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
