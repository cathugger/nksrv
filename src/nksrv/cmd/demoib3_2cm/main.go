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
	"nksrv/lib/captchastore/memstore"
	"nksrv/lib/democonfigs"
	di "nksrv/lib/demoib"
	"nksrv/lib/emime"
	fl "nksrv/lib/filelogger"
	ir "nksrv/lib/ibrouter"
	rj "nksrv/lib/jsonrenderer"
	"nksrv/lib/logx"
	"nksrv/lib/psql"
	"nksrv/lib/psqlib"
	rt "nksrv/lib/tmplrenderer"
	wc "nksrv/lib/webcaptcha"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	httpbind := flag.String("httpbind", "127.0.0.1:1234", "http bind address")
	tmpldir := flag.String("tmpldir", "_demo/tmpl", "template directory")
	readonly := flag.Bool("readonly", false, "read-only mode")
	captchamode := flag.String("captchamode", "simple", "[simple, cookie, ssi, esi]")

	flag.Parse()

	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, logx.DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}
	mlg := logx.NewLogToX(lgr, "main")
	mlg.LogPrint(logx.NOTICE, "initializing")

	err = emime.LoadMIMEDatabase("mime.types")
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "LoadMIMEDatabase err:", err)
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

	var usecookies bool
	var ssi, esi bool
	if !*readonly {
		switch *captchamode {
		case "simple":
			usecookies = false
		case "cookie":
			usecookies = true
		case "ssi":
			ssi = true
		case "esi":
			esi = true
		default:
			panic("unknown captchamode")
		}
	}
	webcap, err := wc.NewWebCaptcha(memstore.NewMemStore(), usecookies)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		return
	}

	psqlibcfg := democonfigs.CfgPSQLIB
	psqlibcfg.DB = &db
	psqlibcfg.Logger = &lgr
	psqlibcfg.WebCaptcha = webcap

	dbib, err := psqlib.NewInitAndPrepare(psqlibcfg)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		return
	}

	rend, err := rt.NewTmplRenderer(dbib, rt.TmplRendererCfg{
		TemplateDir: *tmpldir,
		Logger:      lgr,
		NodeInfo: rt.NodeInfo{
			Captcha: democonfigs.CfgCaptchaInfo,
		},
		WebCaptcha: webcap,
		SSI:        ssi,
		ESI:        esi,
		StaticDir:  di.StaticDir.Dir(),
	})
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
		CaptchaInfo:  democonfigs.CfgCaptchaInfo,
	}
	if !*readonly {
		arcfg.WebPostProvider = dbib
		ircfg.WebPostProvider = dbib
		ircfg.WebCaptcha = webcap
		ircfg.SSI = ssi
		ircfg.ESI = esi
	}
	arh := ar.NewAPIRouter(arcfg)
	ircfg.APIHandler = arh
	irh := ir.NewIBRouter(ircfg)

	server := &http.Server{Addr: *httpbind, Handler: irh}

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
