package main

import (
	"flag"
	"fmt"
	"os"

	"nksrv/lib/app/demo/democonfigs"
	fl "nksrv/lib/utils/logx/filelogger"
	"nksrv/lib/utils/logx"
	"nksrv/lib/app/base/psql"
	"nksrv/lib/app/psqlib"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	modprivs := flag.String("priv", "mod", "priv to evaluate to")

	flag.Parse()

	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, logx.DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}
	mlg := logx.NewLogToX(lgr, "main")

	modpriv, ok := psqlib.StringToModPriv(*modprivs)
	if !ok {
		mlg.LogPrintln(logx.CRITICAL, "unrecognised mod priv %q", modprivs)
		return
	}

	psqlcfg := psql.DefaultConfig
	psqlcfg.Logger = lgr
	psqlcfg.ConnStr = *dbconnstr

	db, err := psql.OpenAndPrepare(psqlcfg)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psql.OpenAndPrepare error:", err)
		return
	}
	defer db.Close()

	psqlibcfg := democonfigs.CfgPSQLIB
	psqlibcfg.DB = &db
	psqlibcfg.Logger = &lgr

	dbib, err := psqlib.NewInitAndPrepare(psqlibcfg)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		return
	}

	args := flag.Args()
	dbib.DemoSetModPriv(args, modpriv)
}
