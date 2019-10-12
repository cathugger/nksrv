package psql

import (
	"os"
	"testing"
	"time"

	fl "nksrv/lib/filelogger"
	"nksrv/lib/logx"
	"nksrv/lib/psql/testutil"
)

func newLogger() (lgr logx.LoggerX) {
	lgr, err := fl.NewFileLogger(os.Stderr, logx.DEBUG, fl.ColorAuto)
	if err != nil {
		panic("fl.NewFileLogger err: " + err.Error())
	}
	return
}

func TestInit(t *testing.T) {
	dbn := testutil.MakeTestDB()
	defer testutil.DropTestDB(dbn)

	lgr := newLogger()

	db, err := OpenAndPrepare(Config{
		ConnStr: "user=" + testutil.TestUser +
			" dbname=" + dbn +
			" host=" + testutil.PSQLHost,
		Logger: lgr,
	})
	if err != nil {
		panic("OAP err: " + err.Error())
	}

	err = db.Close()
	if err != nil {
		panic("close err: " + err.Error())
	}
}

func TestListen(t *testing.T) {
	dbn := testutil.MakeTestDB()
	defer testutil.DropTestDB(dbn)

	lgr := newLogger()

	db, err := OpenPSQL(Config{
		ConnStr: "user=" + testutil.TestUser +
			" dbname=" + dbn +
			" host=" + testutil.PSQLHost,
		Logger: lgr,
	})
	if err != nil {
		panic("Open err: " + err.Error())
	}
	defer func() {
		err = db.Close()
		if err != nil {
			panic("close err: " + err.Error())
		}
	}()

	nok := 0
	x := make(chan string)

	db.Listen("test", func(e string, rst bool) {
		if rst {
			panic("shouldn't have rst but does")
		}
		if e != "test1" {
			panic("wrong e val")
		}
		nok++
		x <- e
	})

	go func() {
		time.Sleep(400 * time.Millisecond)
		_, exx := db.DB.Exec("NOTIFY test, 'test1'")
		if exx != nil {
			panic("exec err: " + exx.Error())
		}
	}()

	select {
	case <-time.After(5 * time.Second):
		panic("timer expired")
	case <-x:
		// OK
	}
	time.Sleep(500 * time.Millisecond)

	if nok != 1 {
		panic("wrong nok val")
	}
}
