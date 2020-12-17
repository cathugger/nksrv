package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/net/proxy"

	"nksrv/lib/app/base/psql"
	"nksrv/lib/app/demo/democonfigs"
	"nksrv/lib/app/demo/demohelper"
	"nksrv/lib/app/psqlib"
	"nksrv/lib/nntp"
	"nksrv/lib/thumbnailer/extthm"
	"nksrv/lib/utils/logx"
	. "nksrv/lib/utils/logx"
	fl "nksrv/lib/utils/logx/filelogger"
	"nksrv/lib/utils/xdialer"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	nntpconn := flag.String("nntpconn", "", "nntp server connect string")
	socks := flag.String("socks", "", "socks proxy address")
	pullkey := flag.String("pullkey", "test", "puller identifier used to store state")
	notrace := flag.Bool("notrace", false, "disable NNTP Path trace")
	thumbext := flag.Bool("extthm", false, "use extthm")
	nodename := flag.String("nodename", "nekochan", "node name. must be non-empty")
	ngp := flag.String("ngp", "*", "new group policy: which groups can be automatically added?")

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
	psqlibcfg.NGPGlobal = *ngp
	if *thumbext {
		psqlibcfg.TBuilder = extthm.DefaultConfig
	}

	dbib, err := psqlib.NewInitAndPrepare(psqlibcfg)
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psqlib.NewInitAndPrepare error:", err)
		return
	}

	dbpuller, err := dbib.NewPullerDB(*pullkey, "", *notrace)
	if err != nil {
		mlg.LogPrintln(CRITICAL, "dbib.NewPullerDB failed:", err)
		return
	}
	dbib.ClearPullerDBs()

	puller := nntp.NewNNTPPuller(dbpuller, lgr)

	d, proto, host, e := xdialer.XDial(*nntpconn)
	if e != nil {
		mlg.LogPrintln(CRITICAL, "dial fail:", e)
		return
	}

	if *socks != "" {
		d, e = proxy.SOCKS5("tcp", *socks, nil, nil)
		if e != nil {
			mlg.LogPrintln(CRITICAL, "SOCKS5 fail:", e)
			return
		}
	}

	mlg.LogPrintf(
		NOTICE, "starting nntp puller with proto(%s) host(%s)", proto, host)
	puller.Run(d, proto, host)
	mlg.LogPrintf(NOTICE, "nntp puller terminated")
}
