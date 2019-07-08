package psqlib

// database stuff

import (
	"context"
	"database/sql"
	"fmt"

	"centpd/lib/sqlbucket"
)

const currIb0Version = "demo6"

func (sp *PSQLIB) doDbIbit() (err error) {
	stmts, err := sqlbucket.LoadFromFile("aux/psqlib/init.sql")
	if err != nil {
		return fmt.Errorf("err on sql loading: %v", err)
	}
	if stmts["version"][0] != currIb0Version {
		return fmt.Errorf("wrong sql file version %v want %v",
			stmts["version"][0], currIb0Version)
	}

	tx, err := sp.db.DB.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if err != nil {
		return fmt.Errorf("err on BeginTx: %v", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for i, s := range stmts["init"] {
		_, err = tx.Exec(s)
		if err != nil {
			err = fmt.Errorf("err on stmt %d: %v", i, err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("err on Commit: %v", err)
	}
	return
}

func (sp *PSQLIB) InitIb0() {
	e := sp.doDbIbit()
	if e != nil {
		panic(e)
	}
}

func (sp *PSQLIB) CheckIb0() (initialised bool, versionerror error) {
	q := "SHOW server_version_num"
	var vernum int64
	err := sp.db.DB.QueryRow(q).Scan(&vernum)
	if err != nil {
		return false, sp.sqlError("server version query", err)
	}
	const verreq = 100000
	if vernum < verreq {
		return false, fmt.Errorf("we require at least server version %d, got %d", verreq, vernum)
	}

	q = "SELECT version FROM capabilities WHERE component = 'ib0' LIMIT 1"
	var ver string
	err = sp.db.DB.QueryRow(q).Scan(&ver)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sp.sqlError("version row query", err)
	}

	if ver != currIb0Version {
		return true, fmt.Errorf("incorrect ib0 schema version: %q (our: %q)", ver, currIb0Version)
	}

	return true, nil
}
