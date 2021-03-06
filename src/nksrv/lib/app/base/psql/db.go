package psql

// generic PSQL connector
// can be used by more concrete forum packages

import (
	"fmt"
	"strings"
)

const currDBVersion = ""

var dbInitStatements = []string{
	`CREATE TABLE capabilities (
	component TEXT NOT NULL PRIMARY KEY,
	version   TEXT NOT NULL
)`,
	`INSERT INTO capabilities(component,version) VALUES ('','')`,
}

func (sp PSQL) InitDB() {
	var gotcs string
	q := `SELECT character_set_name FROM information_schema.character_sets`
	err := sp.DB.QueryRow(q).Scan(&gotcs)
	if err != nil {
		panic(err)
	}
	const wantcs = "UTF8"
	if !strings.EqualFold(gotcs, wantcs) {
		panic(fmt.Errorf(
			`bad database charset: expected %q got %q`, wantcs, gotcs))
	}

	for i := range dbInitStatements {
		sp.DB.MustExec(dbInitStatements[i])
	}
}

func (sp PSQL) IsValidDB() (bool, error) {
	var exists bool
	q := `SELECT EXISTS (
	SELECT 1
	FROM pg_tables
	WHERE schemaname = 'public' AND tablename = 'capabilities'
)`
	err := sp.DB.QueryRow(q).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (sp PSQL) CheckVersion() error {
	var ver string
	q := `SELECT version FROM capabilities WHERE component = '' LIMIT 1`
	err := sp.DB.QueryRow(q).Scan(&ver)
	if err != nil {
		return sp.sqlError("version row query", err)
	}
	if ver != currDBVersion {
		return fmt.Errorf(
			"incorrect database version: %q (our: %q)", ver, currDBVersion)
	}
	return nil
}
