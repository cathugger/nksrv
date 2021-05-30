package pgxhelper

import (
	"fmt"
	"os"
	"testing"

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

func testOK(t *testing.T, which string) {
	db, err := pgxProv.NewDatabase()
	if err != nil {
		t.Errorf("pgxProv.NewDatabase err: %v", err)
		return
	}
	defer func() {
		e := db.Close()
		if e != nil {
			t.Errorf("db.Close err: %v", e)
		}
	}()

	st, err := NewSchemaTool(os.DirFS("testdata/" + which))
	if err != nil {
		t.Errorf("NewSchemaTool err: %v", err)
		return
	}

	err = st.CheckDBConfig(db.Config, "test")
	if err != ErrNeedsInitialization {
		t.Errorf("unexpected CheckDBConfig err: %v", err)
		return
	}

	didSomething, err := st.MigrateDBConfig(db.Config, "test")
	if err != nil {
		t.Errorf("unexpected MigrateDBConfig err: %v", err)
		return
	}
	if !didSomething {
		t.Errorf("didSomething should be true")
		return
	}

	err = st.CheckDBConfig(db.Config, "test")
	if err != nil {
		t.Errorf("unexpected CheckDBConfig err: %v", err)
		return
	}

	didSomething, err = st.MigrateDBConfig(db.Config, "test")
	if err != nil {
		t.Errorf("unexpected MigrateDBConfig err: %v", err)
		return
	}
	if didSomething {
		t.Errorf("didSomething should be false")
		return
	}
}

func TestFS01(t *testing.T) {
	testOK(t, "01")
}

func TestFS02(t *testing.T) {
	testOK(t, "02")
}

func TestFS03(t *testing.T) {
	testOK(t, "03")
}
