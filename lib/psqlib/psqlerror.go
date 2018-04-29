package psqlib

import (
	. "../logx"
	"errors"
	"fmt"
	"runtime/debug"
)

func (sp *PSQLIB) sqlError(when string, err error) error {
	emsg := fmt.Sprintf("error on %s: %v", when, err)
	if sp.log.Level() <= ERROR {
		sp.log.LogPrint(ERROR, emsg)
		if sp.log.LockWrite(ERROR) {
			sp.log.Write(debug.Stack())
			sp.log.Close()
		}
	}
	return errors.New(emsg)
}
