package pibase

// in idea this should be in pibaseweb but since we sorta need to init it from pibase...

import (
	"os"

	"nksrv/lib/fstore"
	"nksrv/lib/mail/form"
)

type FormFileOpener struct {
	*fstore.FStore
}

var _ form.FileOpener = FormFileOpener{}

func (o FormFileOpener) OpenFile() (*os.File, error) {
	return o.FStore.NewFile("tmp", "webpost-", "")
}
