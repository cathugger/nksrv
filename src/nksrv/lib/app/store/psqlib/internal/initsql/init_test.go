package initsql

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/jackc/pgx/v4"

	. "nksrv/lib/app/store/psqlib/internal/basesql"
	"nksrv/lib/app/store/psqlib/internal/sqlcode"
	"nksrv/lib/utils/sqlhelper/pgxhelper"
	"nksrv/lib/utils/testhelper"
)

var pgxProv testhelper.PGXProvider

func TestMain(m *testing.M) {

	code := func() (code int) {
		// we gonna use postgresql stuff
		pp, err := testhelper.NewPGXProvider()
		if err != nil {
			fmt.Fprintf(os.Stderr, "NewPGXProvider err: %v\n", err)
			return 1
		}
		defer func() {
			e := pp.Close()
			if e != nil {
				fmt.Fprintf(os.Stderr, "PGXProvider.Close err: %v\n", e)
				if code == 0 {
					code = 1
				}
			}
		}()
		pgxProv = pp

		return m.Run()
	}()

	os.Exit(code)
}

func testMigrate(t *testing.T, db *pgx.Conn, dir fs.FS) {
	st, err := pgxhelper.NewSchemaTool(dir)
	if err != nil {
		t.Fatalf("pgxhelper.NewSchemaTool err: %v", err)
	}
	didSomething, err := MigrateDBConn(&st, db)
	if err != nil {
		t.Fatalf("MigrateDBConn err: %v", err)
	}
	if !didSomething {
		t.Fatalf("MigrateDBConn err: %v", err)
	}
}

func testPrepare(t *testing.T, dir fs.FS, stmts *[SISize]string) {
	dbHandle, err := pgxProv.NewDatabase()
	if err != nil {
		t.Fatalf("pgxProv.NewDatabase err: %v", err)
	}
	defer dbHandle.Close()

	db, err := pgx.ConnectConfig(context.Background(), dbHandle.Config)
	if err != nil {
		t.Fatalf("pgx.ConnectConfig err: %v", err)
	}
	defer db.Close(context.Background())

	testMigrate(t, db, dir)

	err = PrepareStatementsForConn(context.Background(), db, stmts)
	if err != nil {
		t.Fatalf("PrepareStatementsForConn err: %v", err)
	}
}

func TestEmbed(t *testing.T) {

	stmts, err := LoadStatementsFromFS(sqlcode.Statements)
	if err != nil {
		t.Fatalf("LoadStatementsFromFS err: %v", err)
	}

	testPrepare(t, sqlcode.Schema, stmts)
}

func TestFS(t *testing.T) {

	dir := os.DirFS("../sqlcode")

	stmts, err := LoadStatementsFromFS(dir)
	if err != nil {
		t.Fatalf("LoadStatementsFromFS err: %v", err)
	}

	testPrepare(t, dir, stmts)
}
