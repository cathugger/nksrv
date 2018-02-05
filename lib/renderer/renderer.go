package renderer

import "net/http"

// I guess this could render not only to HTML but to stuff like JSON too
type Renderer interface {
	ServeBoardList(w http.ResponseWriter, r *http.Request)
	ServeThreadListPage(w http.ResponseWriter, r *http.Request, board string, page uint32)
	ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string)
	ServeThread(w http.ResponseWriter, r *http.Request, board, thread string)
}
