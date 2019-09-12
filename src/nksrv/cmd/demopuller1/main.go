package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"nksrv/lib/democonfigs"
	"nksrv/lib/emime"
	fl "nksrv/lib/filelogger"
	"nksrv/lib/logx"
	. "nksrv/lib/logx"
	"nksrv/lib/nntp"
	"nksrv/lib/psql"
	"nksrv/lib/psqlib"
	"nksrv/lib/thumbnailer/extthm"
	"nksrv/lib/xdialer"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
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

	args := flag.Args()
	if len(args) < 2 {
		mlg.LogPrintln(CRITICAL, "no pull hosts specified")
		return
	}
	if len(args)%2 != 0 {
		mlg.LogPrintln(CRITICAL, "odd argument count")
		return
	}

	var pullers []*nntp.NNTPPuller
	for i := 0; i+1 < len(args); i += 2 {
		key := args[i]
		notrace := false
		if key[0] == '.' {
			key = key[1:]
			notrace = true
		}

		dbpuller, err := dbib.NewPullerDB(key, "", notrace)
		if err != nil {
			mlg.LogPrintln(CRITICAL, "dbib.NewPullerDB failed:", err)
			return
		}
		puller := nntp.NewNNTPPuller(dbpuller, lgr)

		pullers = append(pullers, puller)
	}
	dbib.ClearPullerDBs()

	j := 0
	for i := 0; i+1 < len(args); i += 2 {
		addr := args[i+1]

		d, proto, host, e := xdialer.XDial(addr)
		if e != nil {
			mlg.LogPrintf(CRITICAL, "dial %d fail: %v", j, e)
			return
		}

		go func(x int) {
			mlg.LogPrintf(
				NOTICE, "starting nntp puller no. %d with proto(%s) host(%s)", x, proto, host)
			pullers[x].Run(d, proto, host)
			mlg.LogPrintf(NOTICE, "nntp puller no. %d terminated", x)
		}(j)

		j++
	}

	// XXX
	for {
		time.Sleep(30 * time.Second)
	}
}
