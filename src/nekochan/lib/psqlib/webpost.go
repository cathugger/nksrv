package psqlib

import (
	"net/http"
	"os"

	"nekochan/lib/fstore"
	"nekochan/lib/mail/form"
)

type formFileOpener struct {
	*fstore.FStore
}

var _ form.FileOpener = formFileOpener{}

func (o formFileOpener) OpenFile() (*os.File, error) {
	return o.FStore.TempFile("webpost-", "")
}

// FIXME: this probably in future should go thru some sort of abstractation

func (sp *PSQLIB) GetPostParams() (*form.ParserParams, form.FileOpener) {
	return &sp.fpp, sp.ffo
}

func (sp *PSQLIB) PostNewThread(w http.ResponseWriter, r *http.Request, f form.Form,
	board string) {

}

func (sp *PSQLIB) PostNewReply(w http.ResponseWriter, r *http.Request, f form.Form,
	board, thread string) {

}
