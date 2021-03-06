package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	ar "nksrv/lib/app/base/apirouter"
	ir "nksrv/lib/app/base/ibrouter"
	"nksrv/lib/app/base/psql"
	wc "nksrv/lib/app/base/webcaptcha"
	"nksrv/lib/app/captchastore"
	"nksrv/lib/app/captchastore/memstore"
	"nksrv/lib/app/captchastore/psqlstore"
	"nksrv/lib/app/demo/democonfigs"
	"nksrv/lib/app/demo/demohelper"
	di "nksrv/lib/app/demo/demoib"
	"nksrv/lib/app/psqlib"
	rj "nksrv/lib/app/renderer/jsonrenderer"
	rt "nksrv/lib/app/renderer/tmplrenderer"
	"nksrv/lib/thumbnailer/extthm"
	"nksrv/lib/utils/logx"
	fl "nksrv/lib/utils/logx/filelogger"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	httpbind := flag.String("httpbind", "127.0.0.1:1234", "http bind address")
	tmpldir := flag.String("tmpldir", "_demo/tmpl", "template directory")
	readonly := flag.Bool("readonly", false, "read-only mode")
	captchamode := flag.String("captchamode", "simple", "[simple, cookie, ssi, esi]")
	usememstore := flag.Bool("memstore", false, "use memstore instead of psqlstore")
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
	mlg.LogPrint(logx.NOTICE, "initializing")

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

	var cs captchastore.CaptchaStore
	if !*usememstore {
		cs, err = psqlstore.NewInitAndPrepare(&db, lgr)
		if err != nil {
			mlg.LogPrintln(logx.CRITICAL, "psqlstore.NewInitAndPrepare error:", err)
			return
		}
	} else {
		cs = memstore.NewMemStore()
	}

	webcap, err := wc.NewWebCaptcha(cs, usecookies, 60*60)
	if err != nil {
		mlg.LogPrintln(logx.CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		return
	}

	psqlibcfg := democonfigs.CfgPSQLIB
	psqlibcfg.DB = &db
	psqlibcfg.Logger = &lgr
	psqlibcfg.WebCaptcha = webcap
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
		NodeInfo: rt.NodeInfo{
			Captcha: democonfigs.CfgCaptchaInfo,
		},
		WebCaptcha: webcap,
		SSI:        ssi,
		ESI:        esi,
		StaticDir:  di.StaticDir.Dir(),
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
