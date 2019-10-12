package testutil

import (
	crand "crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"os"
)

var (
	PSQLHost = envOrDefault("PG_HOST", "/var/run/postgresql")

	AdmUser = envOrDefault("PG_ADMIN_USER", "postgres")
	AdmDB   = envOrDefault("PG_ADMIN_DB", "postgres")

	TestUser = envOrDefault("PG_TEST_USER", "tester0")
)

func envOrDefault(v, d string) (s string) {
	s = os.Getenv(v)
	if s == "" {
		s = d
	}
	return
}

func testDBName() string {
	nBig, err := crand.Int(crand.Reader, big.NewInt(0x3FffFFffFFffFFff+1))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("nksrv_test_%d", nBig.Int64())
}

func ExecAdmCmd(cmd string) {
	db, err := sql.Open(
		"postgres",
		"user="+AdmUser+" dbname="+AdmDB+" host="+PSQLHost)
	if err != nil {
		panic("sql.Open err: " + err.Error())
	}
	defer db.Close()

	_, err = db.Exec(cmd)
	if err != nil {
		panic("db.Exec err: " + err.Error())
	}
}

func MakeTestDB() (testdb string) {

	testdb = testDBName()

	ExecAdmCmd(fmt.Sprintf(
		"CREATE DATABASE %s OWNER %s ENCODING 'UTF8'",
		testdb, TestUser))

	return
}

func DropTestDB(n string) {
	ExecAdmCmd(fmt.Sprintf("DROP DATABASE %s", n))
}
