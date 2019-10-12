package psql

import (
	"crypto/rand"
	crand "crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	fl "nksrv/lib/filelogger"
	"nksrv/lib/logx"
)

func envOrDefault(v, d string) (s string) {
	s = os.Getenv(v)
	if s == "" {
		s = d
	}
	return
}

var (
	psqlHost = envOrDefault("PG_HOST", "/var/run/postgresql")

	admUser = envOrDefault("PG_ADMIN_USER", "postgres")
	admDB   = envOrDefault("PG_ADMIN_DB", "postgres")

	testUser = envOrDefault("PG_TEST_USER", "tester0")
)

func testDBName() string {
	nBig, err := crand.Int(rand.Reader, big.NewInt(0x3FffFFffFFffFFff+1))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("nksrv_test_%d", nBig.Int64())
}

func execAdmCmd(cmd string) {
	db, err := sql.Open(
		"postgres",
		"user="+admUser+" dbname="+admDB+" host="+psqlHost)
	if err != nil {
		panic("sql.Open err: " + err.Error())
	}
	defer db.Close()

	_, err = db.Exec(cmd)
	if err != nil {
		panic("db.Exec err: " + err.Error())
	}
}

func makeTestDB() (testdb string) {

	testdb = testDBName()

	execAdmCmd(fmt.Sprintf(
		"CREATE DATABASE %s OWNER %s ENCODING 'UTF8'",
		testdb, testUser))

	return
}

func dropTestDB(n string) {
	execAdmCmd(fmt.Sprintf("DROP DATABASE %s", n))
}

func TestInit(t *testing.T) {
	dbn := makeTestDB()
	defer dropTestDB(dbn)

	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, logx.DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}

	db, err := OpenAndPrepare(Config{
		ConnStr: "user=" + testUser + " dbname=" + dbn + " host=" + psqlHost,
		Logger:  lgr,
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
	dbn := makeTestDB()
	defer dropTestDB(dbn)

	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, logx.DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}

	db, err := OpenAndPrepare(Config{
		ConnStr: "user=" + testUser + " dbname=" + dbn + " host=" + psqlHost,
		Logger:  lgr,
	})
	if err != nil {
		panic("OAP err: " + err.Error())
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
