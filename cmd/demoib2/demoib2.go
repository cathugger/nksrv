package main

import (
	di "../../lib/demoib"
	ir "../../lib/ibrouter"
	rt "../../lib/tmplrenderer"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	rend, err := rt.NewTmplRenderer(di.IBProviderDemo{}, "_demo/tmpl")
	if err != nil {
		fmt.Fprintf(os.Stderr, "newtmplrenderer error: %v\n", err)
		return
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
		fmt.Fprintf(os.Stderr, "error from ListenAndServe: %v\n", err)
	}
}
