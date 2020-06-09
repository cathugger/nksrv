package pibase

import (
	"nksrv/lib/fstore"
	"nksrv/lib/mail/form"
	"os"
)

type formFileOpener struct {
	*fstore.FStore
}

var _ form.FileOpener = formFileOpener{}

func (o formFileOpener) OpenFile() (*os.File, error) {
	return o.FStore.NewFile("tmp", "webpost-", "")
}
