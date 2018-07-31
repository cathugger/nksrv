package psqlib

import (
	"database/sql"
	"net/http"
	"os"

	xtypes "github.com/jmoiron/sqlx/types"

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

var FileFields = []string{
	"file", "file2", "file3", "file4",
	"file5", "file6", "file7", "file8",
	"file9", "file10", "file11", "file12",
	"file13", "file14", "file15", "file16",
}

func (sp *PSQLIB) applyInstanceThreadLimits(
	battrib *boardAttributes,
	board string, r *http.Request) {
	// TODO
}

func (sp *PSQLIB) PostNewThread(
	w http.ResponseWriter, r *http.Request, f form.Form,
	board string) (
	error, int) {
	var err error

	// XXX more fields
	if len(f.Values["title"]) != 1 ||
		len(f.Values["message"]) != 1 {
		return errInvalidSubmission, http.StatusBadRequest
	}

	// get info about board, its limits and shit. does it even exists?
	var bid boardID
	var jcfg xtypes.JSONText

	err = sp.db.DB.
		QueryRow("SELECT bid,attrib FROM ib0.boards WHERE bname=$1", board).
		Scan(&bid, &jcfg)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoSuchBoard, http.StatusNotFound
		}
		return sp.sqlError("boards row query scan", err), http.StatusInternalServerError
	}

	battrs := defaultBoardAttributes
	err = jcfg.Unmarshal(&battrs)
	if err != nil {
		return sp.sqlError("board attr json unmarshal", err), http.StatusInternalServerError
	}

	// apply instance-specific limit tweaks
	sp.applyInstanceThreadLimits(&battrs, board, r)
	// TODO actually check for them

	//ftitle := f.Values["title"][0]
	//fmessage := f.Values["message"][0]
	// TODO check for limits

	return nil, 0
}

func (sp *PSQLIB) PostNewReply(
	w http.ResponseWriter, r *http.Request, f form.Form,
	board, thread string) {

}
