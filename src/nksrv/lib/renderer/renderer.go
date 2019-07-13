package renderer

import (
	"net/http"

	ib0 "nksrv/lib/webib0"
)

// this should render only HTML
// API stuff should have different interface
type Renderer interface {
	ServeBoardList(w http.ResponseWriter, r *http.Request)
	ServeThreadListPage(
		w http.ResponseWriter, r *http.Request, board string, page uint32)
	ServeOverboardPage(w http.ResponseWriter, r *http.Request, page uint32)
	ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string)
	ServeThread(w http.ResponseWriter, r *http.Request, board, thread string)

	DressNewBoardResult(
		w http.ResponseWriter, bname string, err error, code int)
	DressPostResult(w http.ResponseWriter, pi ib0.IBPostedInfo,
		newthread bool, err error, code int)

	WebCaptchaInclude(w http.ResponseWriter, r *http.Request)
}
