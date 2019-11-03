package psqlib

import (
	"github.com/lib/pq"

	"nksrv/lib/psql"
)

func (s *PSQLIB) sqlError(when string, err error) error {
	if pqerr, _ := err.(*pq.Error); pqerr != nil && pqerr.Code == "40P01" {
		// deadlock detected
		err = psqlDeadlockError{err}
		// TODO maybe disable logging?
	}
	return psql.SQLError(s.log, when, err)
}
