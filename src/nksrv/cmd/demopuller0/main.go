package main

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"

	"golang.org/x/net/proxy"

	"nksrv/lib/democonfigs"
	"nksrv/lib/emime"
	fl "nksrv/lib/filelogger"
	"nksrv/lib/logx"
	. "nksrv/lib/logx"
	"nksrv/lib/nntp"
	"nksrv/lib/psql"
	"nksrv/lib/psqlib"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	nntpconn := flag.String("nntpconn", "", "nntp server connect string")
	socks := flag.String("socks", "", "socks proxy address")
	pullkey := flag.String("pullkey", "test", "puller identifier used to store state")
	notrace := flag.Bool("notrace", false, "disable NNTP Path trace")

	flag.Parse()

	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		return
	}
	mlg := NewLogToX(lgr, "main")
	mlg.LogPrint(DEBUG, "testing DEBUG log message")
	mlg.LogPrint(INFO, "testing INFO log message")
	mlg.LogPrint(NOTICE, "testing NOTICE log message")
	mlg.LogPrint(WARN, "testing WARN log message")
	mlg.LogPrint(ERROR, "testing ERROR log message")
	mlg.LogPrint(CRITICAL, "testing CRITICAL log message")

	err = emime.LoadMIMEDatabase("mime.types")
	if err != nil {
		mlg.LogPrintln(CRITICAL, "LoadMIMEDatabase err:", err)
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

	dbib, err := psqlib.NewInitAndPrepare(psqlibcfg)
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		return
	}

	dbpuller, err := dbib.NewPullerDB(*pullkey, true, *notrace)
	if err != nil {
		mlg.LogPrintln(CRITICAL, "dbib.NewPullerDB failed:", err)
		return
	}
	dbib.ClearPullerDBs()

	puller := nntp.NewNNTPPuller(dbpuller, lgr)

	var proto, host string
	u, e := url.ParseRequestURI(*nntpconn)
	if e == nil {
		proto, host = u.Scheme, u.Host
	} else {
		proto, host = "tcp", *nntpconn
	}
	if host == "" {
		mlg.LogPrintln(CRITICAL, "no nntpconn host specified")
		return
	}

	var d nntp.Dialer
	if *socks == "" {
		d = &net.Dialer{}
	} else {
		d, e = proxy.SOCKS5("tcp", *socks, nil, nil)
		if e != nil {
			mlg.LogPrintln(CRITICAL, "SOCKS5 fail:", e)
			return
		}
	}

	/*
		defer func() {
			r := recover()
			mlg.LogPrintf(ERROR, "recover: %v", r)
			if mlg.LockWrite(ERROR) {
				mlg.Write(debug.Stack())
				mlg.Close()
			}
		}()
	*/

	mlg.LogPrintf(
		NOTICE, "starting nntp puller with proto(%s) host(%s)", proto, host)
	puller.Run(d, proto, host)
	mlg.LogPrintf(NOTICE, "nntp puller terminated")
}
