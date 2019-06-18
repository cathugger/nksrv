package main

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"

	"golang.org/x/net/proxy"

	"centpd/lib/democonfigs"
	"centpd/lib/emime"
	fl "centpd/lib/filelogger"
	"centpd/lib/logx"
	. "centpd/lib/logx"
	"centpd/lib/nntp"
	"centpd/lib/psql"
	"centpd/lib/psqlib"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")

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

		dbpuller, err := dbib.NewPullerDB(key, true, notrace)
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

		var d nntp.Dialer = &net.Dialer{}
		var proto, host string
		u, e := url.ParseRequestURI(addr)
		if e == nil {
			proto, host = u.Scheme, u.Host
		} else {
			proto, host = "tcp", addr
		}
		if proto == "socks" || proto == "socks5" {
			d, e = proxy.SOCKS5("tcp", host, nil, nil)
			if e != nil {
				mlg.LogPrintln(CRITICAL, "SOCKS5 fail:", e)
				return
			}

			nh := u.Path
			u, e = url.ParseRequestURI(nh)
			if e == nil {
				proto, host = u.Scheme, u.Host
			} else {
				proto, host = "tcp", nh
			}
		}
		if host == "" {
			mlg.LogPrintln(CRITICAL, "no host specified")
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
	}
}
