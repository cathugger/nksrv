package webib0

import (
	"net/http"

	"nksrv/lib/mail/form"
	mm "nksrv/lib/minimail"
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

const (
	IBWebFormTextTitle      = "title"
	IBWebFormTextName       = "name"
	IBWebFormTextMessage    = "message"
	IBWebFormTextOptions    = "options"
	IBWebFormTextCaptchaKey = "captcha_key"
	IBWebFormTextCaptchaAns = "captcha_ans"
)

type IBNewBoardInfo struct {
	Name           string `json:"bname"`
	Description    string `json:"bdesc,omitempty"`
	ThreadsPerPage int32  `json:"threads_per_page,omitempty"` // <= 0 - infinite
	MaxActivePages int32  `json:"max_active_pages,omitempty"` // <= 0 - all pages are active
	MaxPages       int32  `json:"max_pages,omitempty"`        // <= 0 - unlimited
	// TODO more fields
}

type WebPostError struct {
	Err  error
	Code int
}

func (e *WebPostError) Error() string { return e.Err.Error() }
func (e *WebPostError) Unwrap() error { return e.Err }

func UnpackWebPostError(err error) (error, int) {
	if wpe, ok := err.(*WebPostError); ok {
		return wpe.Err, wpe.Code
	} else {
		return err, http.StatusInternalServerError
	}
}

type IBWebPostProvider interface {
	IBGetPostParams() (
		*form.ParserParams, form.FileOpener, func(string) bool)

	IBDefaultBoardInfo() IBNewBoardInfo

	IBPostNewBoard(
		w http.ResponseWriter, r *http.Request, bi IBNewBoardInfo) (
		err error)

	IBPostNewThread(
		w http.ResponseWriter, r *http.Request,
		f form.Form, board string) (
		rInfo IBPostedInfo, err error)

	IBPostNewReply(
		w http.ResponseWriter, r *http.Request,
		f form.Form, board, thread string) (
		rInfo IBPostedInfo, err error)

	IBUpdateBoard(
		w http.ResponseWriter, r *http.Request, bi IBNewBoardInfo) (
		err error)

	IBDeleteBoard(
		w http.ResponseWriter, r *http.Request, board string) (
		err error)

	IBDeletePost(
		w http.ResponseWriter, r *http.Request, board, post string) (
		err error)
}
