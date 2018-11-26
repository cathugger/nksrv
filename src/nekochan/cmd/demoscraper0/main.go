package main

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime"

	"golang.org/x/net/proxy"

	di "nekochan/lib/demoib"
	fl "nekochan/lib/filelogger"
	"nekochan/lib/fstore"
	. "nekochan/lib/logx"
	"nekochan/lib/nntp"
	"nekochan/lib/psql"
	"nekochan/lib/psqlib"
)

func main() {
	var errorcode int
	defer os.Exit(errorcode)

	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	nntpconn := flag.String("nntpconn", "", "nntp server connect string")
	socks := flag.String("socks", "", "socks proxy address")
	scrapekey := flag.String("scrapekey", "test", "scraper identifier used to store state")

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

	db, err := psql.OpenPSQL(psql.Config{
		Logger:  lgr,
		ConnStr: *dbconnstr,
	})
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psql.OpenPSQL error:", err)
		os.Exit(1)
	}
	defer db.Close()

	valid, err := db.IsValidDB()
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psql.OpenPSQL error:", err)
		errorcode = 1
		runtime.Goexit()
	}
	// if not valid, try to create
	if !valid {
		mlg.LogPrint(NOTICE, "uninitialized PSQL db, attempting to initialize")

		db.InitDB()

		// revalidate
		valid, err = db.IsValidDB()
		if err != nil {
			mlg.LogPrintln(CRITICAL, "second psql.OpenPSQL error:", err)
			errorcode = 1
			runtime.Goexit()
		}
		if !valid {
			mlg.LogPrintln(CRITICAL, "psql.IsValidDB failed second validation")
			errorcode = 1
			runtime.Goexit()
		}
	}

	err = db.CheckVersion()
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psql.CheckVersion: ", err)
		errorcode = 1
		runtime.Goexit()
	}

	dbib, err := psqlib.NewPSQLIB(psqlib.Config{
		DB:         db,
		Logger:     lgr,
		SrcCfg:     fstore.Config{"_demo/demoib0/src"},
		ThmCfg:     fstore.Config{"_demo/demoib0/thm"},
		NNTPFSCfg:  fstore.Config{"_demo/demoib0/nntp"},
		AltThumber: di.DemoAltThumber{},
	})
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psqlib.NewPSQLIB error:", err)
		errorcode = 1
		runtime.Goexit()
	}

	valid, err = dbib.CheckIb0()
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psqlib.CheckIb0:", err)
		errorcode = 1
		runtime.Goexit()
	}
	if !valid {
		mlg.LogPrint(NOTICE, "uninitialized PSQLIB db, attempting to initialize")

		dbib.InitIb0()

		valid, err = dbib.CheckIb0()
		if err != nil {
			mlg.LogPrintln(CRITICAL, "second psqlib.CheckIb0:", err)
			errorcode = 1
			runtime.Goexit()
		}
		if !valid {
			mlg.LogPrintln(CRITICAL, "psqlib.CheckIb0 failed second validation")
			errorcode = 1
			runtime.Goexit()
		}
	}

	dbscraper, err := dbib.NewScraperDB(*scrapekey, true)
	if err != nil {
		mlg.LogPrintln(CRITICAL, "dbib.NewScraperDB failed:", err)
		errorcode = 1
		runtime.Goexit()
	}
	dbib.ClearScraperDBs()

	scraper := nntp.NewNNTPScraper(dbscraper, lgr)

	var proto, host string
	u, e := url.ParseRequestURI(*nntpconn)
	if e == nil {
		proto, host = u.Scheme, u.Host
	} else {
		proto, host = "tcp", *nntpconn
	}
	if host == "" {
		mlg.LogPrintln(CRITICAL, "no nntpconn host specified")
		errorcode = 1
		runtime.Goexit()
	}

	var d nntp.Dialer
	if *socks == "" {
		d = &net.Dialer{}
	} else {
		d, e = proxy.SOCKS5("tcp", *socks, nil, nil)
		if e != nil {
			mlg.LogPrintln(CRITICAL, "SOCKS5 fail:", e)
			errorcode = 1
			runtime.Goexit()
		}
	}

	/*
		defer func() {
			r := recover()
			mlg.LogPrintf(ERROR, "recover: %v", r)
			if mlg.LockWrite(ERROR) {
				mlg.Write(debug.Stack())
				mlg.Close()
			}
		}()
	*/

	mlg.LogPrintf(
		NOTICE, "starting nntp scraper with proto(%s) host(%s)", proto, host)
	scraper.Run(d, proto, host)
	mlg.LogPrintf(NOTICE, "nntp scraper terminated")
}
