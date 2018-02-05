package httpibfileprovider

import "net/http"

type HTTPFileProvider interface {
	// caller should not pass id starting with "_tmp/"
	ServeSrc(w http.ResponseWriter, r *http.Request, id string) // original files
	ServeThm(w http.ResponseWriter, r *http.Request, id string) // thumbnails
}
