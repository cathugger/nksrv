package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"nksrv/lib/democonfigs"
	"nksrv/lib/emime"
	fl "nksrv/lib/filelogger"
	"nksrv/lib/logx"
	. "nksrv/lib/logx"
	"nksrv/lib/nntp"
	"nksrv/lib/psql"
	"nksrv/lib/psqlib"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	nntpbind := flag.String("nntpbind", "", "nntp server bind string")

	flag.Parse()

	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}
	mlg := NewLogToX(lgr, "main")
	mlg.LogPrint(DEBUG, "testing DEBUG log message")
	mlg.LogPrint(INFO, "testing INFO log message")
	mlg.LogPrint(NOTICE, "testing NOTICE log message")
	mlg.LogPrint(WARN, "testing WARN log message")
	mlg.LogPrint(ERROR, "testing ERROR log message")
	mlg.LogPrint(CRITICAL, "testing CRITICAL log message")

	err = emime.LoadMIMEDatabase("mime.types")
	if err != nil {
		mlg.LogPrintln(CRITICAL, "LoadMIMEDatabase err:", err)
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
		mlg.LogPrintln(CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		return
	}

	srv := nntp.NewNNTPServer(dbib, lgr, &nntp.DefaultNNTPServerRunCfg)

	var proto, host string
	u, e := url.ParseRequestURI(*nntpbind)
	if e == nil {
		proto, host = u.Scheme, u.Host
	} else {
		proto, host = "tcp", *nntpbind
	}

	// graceful shutdown by signal
	killc := make(chan os.Signal, 2)
	signal.Notify(killc, os.Interrupt, syscall.SIGTERM)
	go func(c chan os.Signal) {
		for {
			s := <-c
			switch s {
			case os.Interrupt, syscall.SIGTERM:
				signal.Reset(os.Interrupt, syscall.SIGTERM)
				fmt.Fprintf(os.Stderr, "killing server\n")
				if srv != nil {
					srv.Close()
				}
				return
			}
		}
	}(killc)

	mlg.LogPrintf(
		NOTICE, "starting nntp server on proto(%s) host(%s)", proto, host)
	err = srv.ListenAndServe(proto, host, nntp.ListenParam{})
	if err != nil {
		mlg.LogPrintf(ERROR, "ListenAndServe returned: %v", err)
	}
}
