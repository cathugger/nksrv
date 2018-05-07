package main

import (
	"context"
	"fmt"
	di "nekochan/lib/demoib"
	ir "nekochan/lib/ibrouter"
	rs "nekochan/lib/rendererstatic"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	rcfg := ir.Cfg{
		HTMLRenderer: rs.RendererStatic{},
		FileProvider: di.IBProviderDemo{},
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

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "error from ListenAndServe: %v\n", err)
	}
}
