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

	"centpd/lib/altthumber"
	ar "centpd/lib/apirouter"
	di "centpd/lib/demoib"
	"centpd/lib/demousrdb"
	fl "centpd/lib/filelogger"
	"centpd/lib/fstore"
	ir "centpd/lib/ibrouter"
	rj "centpd/lib/jsonrenderer"
	"centpd/lib/logx"
	"centpd/lib/oauth2"
	"centpd/lib/psql"
	"centpd/lib/psqlib"
)

func main() {
	var errorcode int
	defer os.Exit(errorcode)

	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	httpbind := flag.String("httpbind", "127.0.0.1:1234", "http bind address")

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

	psqlcfg := psql.DefaultConfig
	psqlcfg.Logger = lgr
	psqlcfg.ConnStr = *dbconnstr

	db, err := psql.OpenAndPrepare(psqlcfg)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psql.OpenAndPrepare error:", err)
		return
	}
	defer db.Close()

	altthm := altthumber.AltThumber(di.DemoAltThumber{})

	dbib, err := psqlib.NewInitAndPrepare(psqlib.Config{
		DB:         &db,
		Logger:     &lgr,
		SrcCfg:     &fstore.Config{"_demo/demoib0/src"},
		ThmCfg:     &fstore.Config{"_demo/demoib0/thm"},
		NNTPFSCfg:  &fstore.Config{"_demo/demoib0/nntp"},
		AltThumber: &altthm,
	})
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		errorcode = 1
		runtime.Goexit()
	}

	rend, err := rj.NewJSONRenderer(dbib, rj.Config{Indent: "  "})
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "rj.NewJSONRenderer error:", err)
		errorcode = 1
		runtime.Goexit()
	}

	seckey := []byte("secrettt")
	wpp := oauth2.NewOAuth2Checker(dbib, seckey, demousrdb.DemoUsrDB{})

	ah := ar.NewAPIRouter(ar.Cfg{
		Renderer:        rend,
		WebPostProvider: wpp,
		Auth:            wpp,
	})
	rcfg := ir.Cfg{
		HTMLRenderer:    rend,
		StaticProvider:  di.IBProviderDemo{},
		FileProvider:    di.IBProviderDemo{},
		WebPostProvider: wpp,
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
