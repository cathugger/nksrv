package psqlib

// database stuff

import (
	"context"
	"database/sql"
	"fmt"

	"nksrv/lib/sqlbucket"
)

const currDbVersion = "demo7"

func (sp *PSQLIB) InitDB() (err error) {

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

	initfs := [...]string{"", "_jobstate", "_puller", "_triggers"}
	for i := range initfs {
		fn := "aux/psqlib/init" + initfs[i] + ".sql"

		stmts, ee := sqlbucket.LoadFromFile(fn)
		if ee != nil {
			err = fmt.Errorf("err on loading %q: %v", fn, ee)
			return
		}

		if i == 0 {
			fvr := stmts["version"][0]
			if fvr != currDbVersion {
				err = fmt.Errorf(
					"wrong sql file version %v want %v",
					fvr, currDbVersion)
				return
			}
			q := `INSERT INTO capabilities(component,version) VALUES ('ib0',$1)`
			_, err = tx.Exec(q, currDbVersion)
			if err != nil {
				return fmt.Errorf("err on version stmt: %v", err)
			}
		}

		for j, s := range stmts["init"+initfs[i]] {
			_, err = tx.Exec(s)
			if err != nil {
				return fmt.Errorf("err on stmt %d: %v", j, err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("err on Commit: %v", err)
	}
	return
}

func (sp *PSQLIB) CheckDB() (initialised bool, versionerror error) {
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

	if ver != currDbVersion {
		return true, fmt.Errorf("incorrect ib0 schema version: %q (our: %q)", ver, currDbVersion)
	}

	return true, nil
}
