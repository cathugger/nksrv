package psql

import (
	"runtime/debug"

	"golang.org/x/xerrors"

	. "nksrv/lib/utils/logx"
)

// SQLError logs and formats error message. if l is nil it doesn't log.
func SQLError(l Logger, when string, err error) error {
	if when != "" {
		err = xerrors.Errorf("error on %s: %w", when, err)
	}
	if l != nil && l.Level() <= ERROR {
		l.LogPrint(ERROR, err.Error())
		if l.LockWrite(ERROR) {
			l.Write(debug.Stack())
			l.Close()
		}
	}
	return err
}

func (s PSQL) sqlError(when string, err error) error {
	return SQLError(s.log, when, err)
}
