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
 * src and thm should have infinite caching
 * static should ALSO have infinite caching BUT we should do something like this:
 * /_static/style.css?timestamp
 * this however will need templating engine awareness
 */
var fsdcfg = fsd.Config{CacheControl: "no-cache, must-revalidate"}

var (
	srcServe    = fsd.NewFServeDir("_demo/demoib0/src", fsdcfg)
	thmServe    = fsd.NewFServeDir("_demo/demoib0/thm", fsdcfg)
	staticServe = fsd.NewFServeDir("_demo/demoib0/static", fsdcfg)
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
