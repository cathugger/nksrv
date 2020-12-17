package psqlcom

import (
	"nksrv/lib/app/base/psql"
)

func (s PSQLCOM) sqlError(when string, err error) error {
	return psql.SQLError(s.log, when, err)
}
