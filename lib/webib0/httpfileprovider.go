package webib0

import "net/http"

type HTTPFileProvider interface {
	ServeSrc(w http.ResponseWriter, r *http.Request, id string) // original files
	ServeThm(w http.ResponseWriter, r *http.Request, id string) // thumbnails
}
