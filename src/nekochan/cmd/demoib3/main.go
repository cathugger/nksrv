package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	ar "nekochan/lib/apirouter"
	di "nekochan/lib/demoib"
	fl "nekochan/lib/filelogger"
	"nekochan/lib/fstore"
	ir "nekochan/lib/ibrouter"
	rj "nekochan/lib/jsonrenderer"
	"nekochan/lib/logx"
	"nekochan/lib/psql"
	"nekochan/lib/psqlib"
)

func main() {
	var errorcode int
	defer os.Exit(errorcode)

	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")

	flag.Parse()

	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, logx.DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}
	mlg := logx.NewLogToX(lgr, "main")
	mlg.LogPrint(logx.DEBUG, "testing DEBUG log message")
	mlg.LogPrint(logx.INFO, "testing INFO log message")
	mlg.LogPrint(logx.NOTICE, "testing NOTICE log message")
	mlg.LogPrint(logx.WARN, "testing WARN log message")
	mlg.LogPrint(logx.ERROR, "testing ERROR log message")
	mlg.LogPrint(logx.CRITICAL, "testing CRITICAL log message")

	db, err := psql.OpenPSQL(psql.Config{
		Logger:  lgr,
		ConnStr: *dbconnstr,
	})
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psql.OpenPSQL error:", err)
		os.Exit(1)
	}
	defer db.Close()

	valid, err := db.IsValidDB()
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psql.OpenPSQL error:", err)
		errorcode = 1
		runtime.Goexit()
	}
	// if not valid, try to create
	if !valid {
		mlg.LogPrint(logx.NOTICE, "uninitialized PSQL db, attempting to initialize")

		db.InitDB()

		// revalidate
		valid, err = db.IsValidDB()
		if err != nil {
			mlg.LogPrintln(logx.CRITICAL, "second psql.OpenPSQL error:", err)
			errorcode = 1
			runtime.Goexit()
		}
		if !valid {
			mlg.LogPrintln(logx.CRITICAL, "psql.IsValidDB failed second validation")
			errorcode = 1
			runtime.Goexit()
		}
	}

	err = db.CheckVersion()
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psql.CheckVersion: ", err)
		errorcode = 1
		runtime.Goexit()
	}

	dbib, err := psqlib.NewPSQLIB(psqlib.Config{
		DB:         db,
		Logger:     lgr,
		SrcCfg:     fstore.Config{"_demo/demoib0/src"},
		ThmCfg:     fstore.Config{"_demo/demoib0/thm"},
		AltThumber: di.DemoAltThumber{},
	})
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.NewPSQLIB error:", err)
		errorcode = 1
		runtime.Goexit()
	}

	valid, err = dbib.CheckIb0()
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.CheckIb0:", err)
		errorcode = 1
		runtime.Goexit()
	}
	if !valid {
		mlg.LogPrint(logx.NOTICE, "uninitialized PSQLIB db, attempting to initialize")

		dbib.InitIb0()

		valid, err = dbib.CheckIb0()
		if err != nil {
			mlg.LogPrintln(logx.CRITICAL, "second psqlib.CheckIb0:", err)
			errorcode = 1
			runtime.Goexit()
		}
		if !valid {
			mlg.LogPrintln(logx.CRITICAL, "psqlib.CheckIb0 failed second validation")
			errorcode = 1
			runtime.Goexit()
		}
	}

	rend, err := rj.NewJSONRenderer(dbib, rj.Config{Indent: "\t"})
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "rj.NewJSONRenderer error:", err)
		errorcode = 1
		runtime.Goexit()
	}
	ah := ar.NewAPIRouter(ar.Cfg{
		Renderer:        rend,
		WebPostProvider: dbib,
	})
	rcfg := ir.Cfg{
		HTMLRenderer:    rend,
		StaticProvider:  di.IBProviderDemo{},
		FileProvider:    di.IBProviderDemo{},
		WebPostProvider: dbib,
		APIHandler:      ah,
	}
	rh := ir.NewIBRouter(rcfg)

	server := &http.Server{Addr: "127.0.0.1:1234", Handler: rh}

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
				if server != nil {
					server.Shutdown(context.Background())
				}
				return
			}
		}
	}(killc)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		mlg.LogPrintln(logx.ERROR, "error from ListenAndServe:", err)
	}
}
