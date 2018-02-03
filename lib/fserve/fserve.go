package fserve

import "net/http"

type FServe interface {
	FServe(w http.ResponseWriter, r *http.Request, id string)
}
