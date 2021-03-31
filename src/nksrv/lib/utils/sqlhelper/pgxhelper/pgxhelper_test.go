package pgxhelper

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4"

	"nksrv/lib/utils/testhelper"
)

var pgxProv testhelper.PGXProvider

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags

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

func TestDB(t *testing.T) {
	db, err := pgxProv.NewDatabase()
	if err != nil {
		t.Errorf("pgxProv.NewDatabase err: %v", err)
		return
	}
	defer func(){
		e := db.Close()
		if e != nil {
			t.Errorf("db.Close err: %v", e)
		}
	}()

	conn, err := pgx.ConnectConfig(context.Background(), db.Config)
	if err != nil {
		t.Errorf("pgx.ConnectConfig err: %v", err)
		return
	}
	defer func(){
		e := conn.Close(context.Background())
		if e != nil {
			t.Errorf("conn.Close err: %v", e)
		}
	}()

	var dst int
	err = conn.QueryRow(context.Background(), "SELECT 1").Scan(&dst)
	if err != nil {
		t.Errorf("conn.QueryRow err: %v", err)
		return
	}
	if dst != 1 {
		t.Errorf("conn.QueryRow bogus ret: %d", dst)
		return
	}
}
