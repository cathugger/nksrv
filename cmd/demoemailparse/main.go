package main

import (
	"fmt"
	"io"
	"os"

	"nksrv/lib/mail"
	. "nksrv/lib/utils/logx"
	fl "nksrv/lib/utils/logx/filelogger"
)

func main() {
	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}
	mlg := NewLogToX(lgr, "idiot.demo")

	f, err := os.Open(os.Args[1])
	if err != nil {
		mlg.LogPrintf(CRITICAL,
			"failed to open file: %v", err)
		os.Exit(1)
	}
	defer f.Close()

	mh, err := mail.ReadHeaders(f, 2<<20)
	if err != nil {
		mlg.LogPrintf(CRITICAL,
			"failed reading headers: %v", err)
		return
	}
	defer mh.Close()

	mlg.LogPrintf(INFO, "Headers: %#v", mh.H)
	mlg.LogPrintf(INFO, "Body:")
	mlg.LockWrite(INFO)
	io.Copy(mlg, mh.B)
	mlg.Close()
}
