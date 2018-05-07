package psql

// generic PSQL connector
// can be used by more concrete forum packages

import (
	"fmt"
)

var currDBVersion = ""

var dbInitStatements = []string{
	`CREATE TABLE capabilities (
	component TEXT NOT NULL UNIQUE,
	version   TEXT NOT NULL
)`,
	`INSERT INTO capabilities(component,version) VALUES ('','')`,
}

func (sp PSQL) InitDB() {
	for i := range dbInitStatements {
		sp.DB.MustExec(dbInitStatements[i])
	}
}

func (sp PSQL) IsValidDB() (bool, error) {
	var exists bool
	err := sp.DB.
		QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'capabilities')`).
		Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (sp PSQL) CheckVersion() error {
	var ver string
	err := sp.DB.
		QueryRow("SELECT version FROM capabilities WHERE component = '' LIMIT 1").
		Scan(&ver)
	if err != nil {
		return sp.sqlError("version row query", err)
	}
	if ver != currDBVersion {
		return fmt.Errorf("incorrect database version: %q (our: %q)", ver, currDBVersion)
	}
	return nil
}
