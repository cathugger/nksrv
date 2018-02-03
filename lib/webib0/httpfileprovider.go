package webib0

import "net/http"

type HTTPFileProvider interface {
	ServeSrc(w http.ResponseWriter, r *http.Request, id string)
	ServeThm(w http.ResponseWriter, r *http.Request, id string)
}
