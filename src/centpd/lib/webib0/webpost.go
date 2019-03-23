package webib0

import (
	"net/http"

	"centpd/lib/mail/form"
	mm "centpd/lib/minimail"
)

type IBPostedInfo struct {
	Board     string          `json:"board"`
	ThreadID  string          `json:"thread_id"`
	PostID    string          `json:"post_id"`
	MessageID mm.CoreMsgIDStr `json:"msgid"` // XXX will we actually use this for anything??
}

var IBWebFormFileFields = []string{
	"file", "file2", "file3", "file4",
	"file5", "file6", "file7", "file8",
	"file9", "file10", "file11", "file12",
	"file13", "file14", "file15", "file16",
}

var (
	IBWebFormTextTitle   = "title"
	IBWebFormTextName    = "name"
	IBWebFormTextMessage = "message"
	IBWebFormTextOptions = "options"
)

type IBNewBoardInfo struct {
	Name           string `json:"bname"`
	Description    string `json:"bdesc,omitempty"`
	ThreadsPerPage int32  `json:"threads_per_page,omitempty"` // <= 0 - infinite
	MaxActivePages int32  `json:"max_active_pages,omitempty"` // <= 0 - all pages are active
	MaxPages       int32  `json:"max_pages,omitempty"`        // <= 0 - unlimited
	// TODO more fields
}

type IBWebPostProvider interface {
	IBGetPostParams() (*form.ParserParams, form.FileOpener)

	IBDefaultBoardInfo() IBNewBoardInfo
	IBPostNewBoard(r *http.Request, bi IBNewBoardInfo) (err error, code int)

	IBPostNewThread(
		r *http.Request, f form.Form, board string) (
		rInfo IBPostedInfo, err error, code int)

	IBPostNewReply(
		r *http.Request, f form.Form, board, thread string) (
		rInfo IBPostedInfo, err error, code int)

	IBUpdateBoard(r *http.Request, bi IBNewBoardInfo) (err error, code int)

	IBDeleteBoard(r *http.Request, board string) (err error, code int)

	IBDeletePost(
		r *http.Request, board, post string) (err error, code int)
}
