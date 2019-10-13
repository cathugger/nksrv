package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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
	"nksrv/lib/thumbnailer/extthm"
	rt "nksrv/lib/tmplrenderer"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	httpbind := flag.String("httpbind", "127.0.0.1:1234", "http bind address")
	tmpldir := flag.String("tmpldir", "_demo/tmpl", "template directory")
	readonly := flag.Bool("readonly", false, "read-only mode")
	thumbext := flag.Bool("extthm", false, "use extthm")
	nodename := flag.String("nodename", "nekochan", "node name. must be non-empty")

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
	psqlibcfg.NodeName = *nodename
	if *thumbext {
		psqlibcfg.TBuilder = extthm.DefaultConfig
	}

	dbib, err := psqlib.NewInitAndPrepare(psqlibcfg)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		return
	}

	tmplrendcfg := rt.TmplRendererCfg{
		TemplateDir: *tmpldir,
		Logger:      lgr,
		StaticDir:   di.StaticDir.Dir(),
	}

	rend, err := rt.NewTmplRenderer(dbib, tmplrendcfg)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "rt.NewTmplRenderer error:", err)
		os.Exit(1)
	}

	jrend, err := rj.NewJSONRenderer(dbib, rj.Config{Indent: "  "})
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "rj.NewJSONRenderer error:", err)
		return
	}
	arcfg := ar.Cfg{
		Renderer: jrend,
	}
	ircfg := ir.Cfg{
		HTMLRenderer: rend,
		StaticDir:    di.StaticDir,
		FileProvider: di.IBProviderDemo{},
	}
	if !*readonly {
		arcfg.WebPostProvider = dbib
		ircfg.WebPostProvider = dbib
	}
	arh := ar.NewAPIRouter(arcfg)
	ircfg.APIHandler = arh
	irh, irh_ctl := ir.NewIBRouter(ircfg)

	server := &http.Server{Addr: *httpbind, Handler: irh}

	// graceful shutdown by signal
	siglist := []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
	killc := make(chan os.Signal, 2)
	signal.Notify(killc, siglist...)
	go func(c chan os.Signal) {
		for {
			s := <-c
			switch s {
			case os.Interrupt, syscall.SIGTERM:
				signal.Reset(siglist...)
				fmt.Fprintf(os.Stderr, "killing server\n")
				if server != nil {
					server.Shutdown(context.Background())
				}
				return
			case syscall.SIGHUP:
				{
					mlg.LogPrintln(logx.NOTICE, "got SIGHUP, will reload templates")
					rend, err := rt.NewTmplRenderer(dbib, tmplrendcfg)
					if err != nil {
						mlg.LogPrintln(logx.ERROR, "rt.NewTmplRenderer error:", err)
						mlg.LogPrintln(logx.NOTICE, "canceling reload because initialization failed")
						break
					}
					irh_ctl.SetHTMLRenderer(rend)
					mlg.LogPrintln(logx.NOTICE, "templates reloaded")
				}
			}
		}
	}(killc)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		mlg.LogPrintln(logx.ERROR, "error from ListenAndServe:", err)
	}
}
