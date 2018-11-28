package staticprovider

import "net/http"

type StaticProvider interface {
	ServeStatic(w http.ResponseWriter, r *http.Request, id string) // static node-specific files
}
