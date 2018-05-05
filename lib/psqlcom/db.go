package psqlcom

import (
	"database/sql"
	"fmt"
)

var currCom0Version = ""

var dbCom0InitStatements = []string{
	`CREATE SCHEMA IF NOT EXISTS com0`,
	`CREATE TABLE com0.groups (
	gname     TEXT  NOT NULL, /* group name */
	bend_type TEXT  NOT NULL, /* backend type */
	bend_cfg  JSONB NOT NULL, /* backend config. format depends on backend type */
	PRIMARY KEY (gname)
)`,
	`INSERT INTO capabilities(component,version) VALUES ('com0','')`,
}

func (sp *PSQLCOM) InitCom0() {
	for i := range dbCom0InitStatements {
		sp.db.DB.MustExec(dbCom0InitStatements[i])
	}
}

func (sp *PSQLCOM) CheckCom0() (initialised bool, versionerror error) {
	var ver string
	err := sp.db.DB.
		QueryRow("SELECT version FROM capabilities WHERE component = 'ib0' LIMIT 1").
		Scan(&ver)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sp.sqlError("version row query", err)
	}
	if ver != currCom0Version {
		return true, fmt.Errorf("incorrect ib0 schema version: %q (our: %q)", ver, currCom0Version)
	}
	return true, nil
}
