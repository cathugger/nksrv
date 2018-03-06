package main

import (
	di "../../lib/demoib"
	fl "../../lib/filelogger"
	ir "../../lib/ibrouter"
	"../../lib/logx"
	rt "../../lib/tmplrenderer"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var err error
	lgr, err := fl.NewFileLogger(os.Stderr, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}
	mlg := logx.NewLogToX(lgr, "main")
	mlg.LogPrintln(logx.DEBUG, "testing DEBUG log message")
	mlg.LogPrintln(logx.VERBOSE, "testing VERBOSE log message")
	mlg.LogPrintln(logx.INFO, "testing INFO log message")
	mlg.LogPrintln(logx.WARN, "testing WARN log message")
	mlg.LogPrintln(logx.ERROR, "testing ERROR log message")
	mlg.LogPrintln(logx.FATAL, "testing FATAL log message")

	rend, err := rt.NewTmplRenderer(di.IBProviderDemo{}, rt.TmplRendererCfg{
		TemplateDir: "_demo/tmpl",
		Logger:      lgr,
	})
	if err != nil {
		mlg.LogPrintln(logx.FATAL, "rt.NewTmplRenderer error:", err)
		os.Exit(1)
	}
	rcfg := ir.Cfg{
		HTMLRenderer:   rend,
		StaticProvider: di.IBProviderDemo{},
		FileProvider:   di.IBProviderDemo{},
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
