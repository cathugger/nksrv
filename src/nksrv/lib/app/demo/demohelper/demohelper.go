package demohelper

import (
	"errors"

	"nksrv/lib/utils/emime"
)

func LoadMIMEDB() (err error) {
	ok, err := emime.LoadMIMEDatabase("mime.types")
	if err != nil {
		return
	}
	if !ok {
		err = errors.New("didn't find mime.types")
	}
	return
}
