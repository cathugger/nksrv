package pibase

import (
	"github.com/lib/pq"

	"nksrv/lib/psql"
)

func (s *PSQLIB) SQLError(when string, err error) error {
	if pqerr, _ := err.(*pq.Error); pqerr != nil {
		switch pqerr.Code {
		case "40001" /* serialization_failure */ :
			err = PSQLRetriableError{err}
		case "40P01" /* deadlock_detected */ :
			err = PSQLRetriableError{err}
		default:
			return psql.SQLError(s.log, when, err)
		}
		// do not log backtrace if we hit expected retriable error
		return psql.SQLError(nil, when, err)
	}
	return psql.SQLError(s.log, when, err)
}

// indicates that psql error is deadlock
type PSQLRetriableError struct {
	error
}

func (x PSQLRetriableError) Unwrap() error { return x.error }
