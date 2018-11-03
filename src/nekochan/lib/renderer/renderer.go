package renderer

import (
	"net/http"

	ib0 "nekochan/lib/webib0"
)

// this should render only HTML
// API stuff should have different interface
type Renderer interface {
	ServeBoardList(w http.ResponseWriter, r *http.Request)
	ServeThreadListPage(
		w http.ResponseWriter, r *http.Request, board string, page uint32)
	ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string)
	ServeThread(w http.ResponseWriter, r *http.Request, board, thread string)

	DressNewBoardResult(w http.ResponseWriter, err error, code int)
	DressPostResult(w http.ResponseWriter, pi ib0.IBPostedInfo,
		newthread bool, err error, code int)
}
