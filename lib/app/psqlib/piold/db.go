package psqlib

// database stuff

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/lib/pq"

	"nksrv/lib/utils/sqlbucket"
)

const currDbVersion = "demo8"

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

	initfs := [...]string{
		"", "_jobstate", "_puller",
		"_triggers",
		"_triggers_banlist",
		"_triggers_boards",
		"_triggers_bposts",
		"_triggers_files",
		"_triggers_gposts",
		"_triggers_modlist",
		"_triggers_modsets",
		"_triggers_threads",
	}
	for i := range initfs {
		fn := "etc/psqlib/init" + initfs[i] + ".sql"

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

		stmtn := "init" + initfs[i]
		stmtss := stmts[stmtn]
		for j, s := range stmtss {
			_, err = tx.Exec(s)
			if err != nil {

				if pe, _ := err.(*pq.Error); pe != nil {

					pos, _ := strconv.Atoi(pe.Position)
					ss, se := pos, pos
					for ss > 0 && s[ss-1] != '\n' {
						ss--
					}
					for se < len(s) && s[se] != '\n' {
						se++
					}

					return fmt.Errorf(
						"err on %s stmt %d pos[%s] msg[%s] detail[%s] line[%s]\nstmt:\n%s",
						stmtn, j, pe.Position, pe.Message, pe.Detail, s[ss:se], s)
				}

				return fmt.Errorf("err on %s stmt %d: %v", stmtn, j, err)

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
		return false, sp.SQLError("server version query", err)
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
		return false, sp.SQLError("version row query", err)
	}

	if ver != currDbVersion {
		return true, fmt.Errorf("incorrect ib0 schema version: %q (our: %q)", ver, currDbVersion)
	}

	return true, nil
}
