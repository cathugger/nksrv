package handler

import "net/http"

// this makes more sense than 404 for fallback
func badRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "400 bad request", http.StatusBadRequest)
}

// this works for method handler
func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
}

// this should work for host handler
func internalServerError(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "500 internal server error", http.StatusInternalServerError)
}
