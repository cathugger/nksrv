package psql

// generic PSQL connector
// can be used by more concrete forum packages

import (
	"fmt"
	"strings"
)

var currDBVersion = ""

var dbInitStatements = []string{
	`CREATE TABLE capabilities (
	component TEXT NOT NULL PRIMARY KEY,
	version   TEXT NOT NULL
)`,
	`INSERT INTO capabilities(component,version) VALUES ('','')`,
}

func (sp PSQL) InitDB() {
	var charset string
	err := sp.DB.
		QueryRow(`SELECT character_set_name FROM information_schema.character_sets`).
		Scan(&charset)
	if err != nil {
		panic(err)
	}
	if !strings.EqualFold(charset, "UTF8") {
		panic(fmt.Errorf(
			"bad database charset: expected \"UTF8\" got %q", charset))
	}

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
