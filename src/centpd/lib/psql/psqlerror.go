package psql

import (
	"errors"
	"fmt"
	"runtime/debug"

	. "centpd/lib/logx"
)

func SQLError(l Logger, when string, err error) error {
	emsg := fmt.Sprintf("error on %s: %v", when, err)
	if l.Level() <= ERROR {
		l.LogPrint(ERROR, emsg)
		if l.LockWrite(ERROR) {
			l.Write(debug.Stack())
			l.Close()
		}
	}
	return errors.New(emsg)
}

func (s PSQL) sqlError(when string, err error) error {
	return SQLError(s.log, when, err)
}
