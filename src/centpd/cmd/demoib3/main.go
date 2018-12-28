package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/lib/pq"
	"github.com/luna-duclos/instrumentedsql"

	"centpd/lib/altthumber"
	ar "centpd/lib/apirouter"
	di "centpd/lib/demoib"
	"centpd/lib/emime"
	fl "centpd/lib/filelogger"
	"centpd/lib/fstore"
	ir "centpd/lib/ibrouter"
	rj "centpd/lib/jsonrenderer"
	"centpd/lib/logx"
	"centpd/lib/psql"
	"centpd/lib/psqlib"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	httpbind := flag.String("httpbind", "127.0.0.1:1234", "http bind address")
	logsql := flag.Bool("logsql", false, "sql logging")

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

	err = emime.LoadMIMEDatabase("mime.types")
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "LoadMIMEDatabase err:", err)
		return
	}

	sqlconf := psql.DefaultConfig
	sqlconf.Logger = lgr
	sqlconf.ConnStr = *dbconnstr

	if *logsql {
		logger := instrumentedsql.LoggerFunc(
			func(ctx context.Context, msg string, keyvals ...interface{}) {
				mlg.LogPrintf(logx.DEBUG, "SQL: %s %v", msg, keyvals)
			})
		const drvstr = "instrumented-postgres"
		sql.Register(drvstr,
			instrumentedsql.WrapDriver(&pq.Driver{},
				/*instrumentedsql.WithTraceRowsNext(),*/
				instrumentedsql.WithLogger(logger),
				instrumentedsql.WithNoTraceRowsNext()))
		sqlconf.ConnDriver = drvstr
	}

	db, err := psql.OpenPSQL(sqlconf)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psql.OpenPSQL error:", err)
		return
	}
	defer db.Close()

	valid, err := db.IsValidDB()
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psql.OpenPSQL error:", err)
		return
	}
	// if not valid, try to create
	if !valid {
		mlg.LogPrint(logx.NOTICE, "uninitialized PSQL db, attempting to initialize")

		db.InitDB()

		// revalidate
		valid, err = db.IsValidDB()
		if err != nil {
			mlg.LogPrintln(logx.CRITICAL, "second psql.OpenPSQL error:", err)
			return
		}
		if !valid {
			mlg.LogPrintln(logx.CRITICAL, "psql.IsValidDB failed second validation")
			return
		}
	}

	err = db.CheckVersion()
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psql.CheckVersion: ", err)
		return
	}

	altthm := altthumber.AltThumber(di.DemoAltThumber{})

	dbib, err := psqlib.NewPSQLIB(psqlib.Config{
		DB:         &db,
		Logger:     &lgr,
		SrcCfg:     &fstore.Config{"_demo/demoib0/src"},
		ThmCfg:     &fstore.Config{"_demo/demoib0/thm"},
		NNTPFSCfg:  &fstore.Config{"_demo/demoib0/nntp"},
		AltThumber: &altthm,
	})
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.NewPSQLIB error:", err)
		return
	}

	valid, err = dbib.CheckIb0()
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.CheckIb0:", err)
		return
	}
	if !valid {
		mlg.LogPrint(logx.NOTICE, "uninitialized PSQLIB db, attempting to initialize")

		dbib.InitIb0()

		valid, err = dbib.CheckIb0()
		if err != nil {
			mlg.LogPrintln(logx.CRITICAL, "second psqlib.CheckIb0:", err)
			return
		}
		if !valid {
			mlg.LogPrintln(logx.CRITICAL, "psqlib.CheckIb0 failed second validation")
			return
		}
	}

	rend, err := rj.NewJSONRenderer(dbib, rj.Config{Indent: "  "})
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "rj.NewJSONRenderer error:", err)
		return
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

	server := &http.Server{Addr: *httpbind, Handler: rh}

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
