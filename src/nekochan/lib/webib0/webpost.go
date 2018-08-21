package webib0

import (
	"net/http"

	"nekochan/lib/mail/form"
)

type IBPostedInfo struct {
	Board     string
	ThreadID  string
	PostID    string
	MessageID string // XXX will we actually use this for anything??
}

var IBWebFormFileFields = []string{
	"file", "file2", "file3", "file4",
	"file5", "file6", "file7", "file8",
	"file9", "file10", "file11", "file12",
	"file13", "file14", "file15", "file16",
}

var (
	IBWebFormTextTitle   = "title"
	IBWebFormTextMessage = "message"
)

type IBWebPostProvider interface {
	IBGetPostParams() (*form.ParserParams, form.FileOpener)
	IBPostNewThread(
		r *http.Request, f form.Form, board string) (
		rInfo IBPostedInfo, err error, _ int)
	IBPostNewReply(
		r *http.Request, f form.Form, board, thread string) (
		rInfo IBPostedInfo, err error, _ int)
}
