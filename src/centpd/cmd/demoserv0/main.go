package main

import (
	"fmt"
	"os"

	fl "centpd/lib/filelogger"
	. "centpd/lib/logx"
	"centpd/lib/nntp"
	nntptest "centpd/lib/nntp/testsrv"
)

func main() {
	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}

	mlg := NewLogToX(lgr, "main")
	mlg.LogPrint(DEBUG, "lol hi")

	prov := &nntptest.TestSrv{
		SupportNewNews:     true,
		SupportOverByMsgID: true,
		SupportHdr:         true,
		SupportIHave:       true,
		SupportPost:        true,
		SupportStream:      true,

		PostingPermit:  true,
		PostingAccept:  true,
		TransferPermit: true,
		TransferAccept: true,

		Log: NewLogToX(lgr, "testsrv"),
	}

	srv := nntp.NewNNTPServer(prov, lgr, &nntp.NNTPServerRunCfg{})

	err = srv.ListenAndServe("tcp4", "127.0.0.1:6633", nntp.ListenParam{})
	if err != nil {
		mlg.LogPrintf(ERROR, "ListenAndServe returned: %v", err)
	}
}
