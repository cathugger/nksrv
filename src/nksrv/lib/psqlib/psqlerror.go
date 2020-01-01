package psqlib

import (
	"github.com/lib/pq"

	"nksrv/lib/psql"
)

func (s *PSQLIB) sqlError(when string, err error) error {
	if pqerr, _ := err.(*pq.Error); pqerr != nil {
		switch pqerr.Code {
			case "40001" /* serialization_failure */ :
				err = psqlRetriableError{err}
			case "40P01" /* deadlock_detected */ :
				err = psqlRetriableError{err}
			default:
				return psql.SQLError(s.log, when, err)
		}
		// do not log backtrace if we hit expected retriable error
		return psql.SQLError(nil, when, err)
	}
	return psql.SQLError(s.log, when, err)
}
