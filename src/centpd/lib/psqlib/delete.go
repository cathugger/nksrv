package psqlib

import (
	"database/sql"
	"errors"
)

func deleteByMsgID(tx *sql.Tx, cmsgids CoreMsgIDStr) (err error) {
	// TODO
	return errors.New("unimplemented")
}
