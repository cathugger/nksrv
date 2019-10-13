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

	ar "nksrv/lib/apirouter"
	"nksrv/lib/democonfigs"
	"nksrv/lib/demohelper"
	di "nksrv/lib/demoib"
	fl "nksrv/lib/filelogger"
	ir "nksrv/lib/ibrouter"
	rj "nksrv/lib/jsonrenderer"
	"nksrv/lib/logx"
	"nksrv/lib/psql"
	"nksrv/lib/psqlib"
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

	err = demohelper.LoadMIMEDB()
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "LoadMIMEDB err:", err)
		return
	}

	sqlcfg := psql.DefaultConfig
	sqlcfg.Logger = lgr
	sqlcfg.ConnStr = *dbconnstr

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
		sqlcfg.ConnDriver = drvstr
	}

	db, err := psql.OpenAndPrepare(sqlcfg)
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
		StaticDir:       di.StaticDir,
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
