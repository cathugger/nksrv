package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"nksrv/lib/demohelper"
	"nksrv/lib/emime"
	fl "nksrv/lib/filelogger"
	"nksrv/lib/fstore"
	. "nksrv/lib/logx"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/thumbnailer/extthm"
	"nksrv/lib/thumbnailer/gothm"
)

func main() {

	thumbdir := flag.String("thumbdir", "_demothm", "thumbnail directory")
	thumbcolor := flag.String("color", "", "background color")
	thumbgray := flag.Bool("grayscale", false, "grayscale")
	thumbext := flag.Bool("extthm", false, "use extthm")

	flag.Parse()

	lgr, err := fl.NewFileLogger(os.Stderr, DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}

	fs, err := fstore.OpenFStore(fstore.Config{Path: *thumbdir})
	if err != nil {
		fmt.Printf("err opening fstore: %v\n", err)
		return
	}

	var thm thumbnailer.Thumbnailer
	if !*thumbext {
		thm, err = gothm.DefaultConfig.BuildThumbnailer(&fs, lgr)
	} else {
		thm, err = extthm.DefaultConfig.BuildThumbnailer(&fs, lgr)
	}
	if err != nil {
		fmt.Printf("err building thumbnailer: %v\n", err)
		return
	}

	err = demohelper.LoadMIMEDB()
	if err != nil {
		fmt.Printf("err loading mime db: %v\n", err)
		return
	}

	tcfg := thumbnailer.ThumbConfig{
		Width:       200,
		Height:      200,
		AudioWidth:  350,
		AudioHeight: 350,
		Color:       *thumbcolor,
		Grayscale:   *thumbgray,
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.PrintDefaults()
		return
	}
	for _, arg := range args {
		f, err := os.Open(arg)
		if err != nil {
			fmt.Printf("err opening %q: %v\n", arg, err)
			return
		}

		base := path.Base(arg)
		var ext string
		if i := strings.LastIndexByte(base, '.'); i >= 0 {
			ext = base[i+1:]
		}

		mtype := emime.MIMETypeByExtension(ext)

		fmt.Printf("processing %q (ext %q mime %q)...\n", arg, ext, mtype)

		res, fi, err := thm.ThumbProcess(f, ext, mtype, tcfg)
		if err != nil {
			fmt.Printf("err thumbnailing: %v", err)
			return
		}
		if res.FileName == "" {
			fmt.Printf("thumbnailer didn't work for this file\n")
			continue
		}

		fmt.Printf("thumbnailed %q, res %#v, fi %#v\n", base, res, fi)

		to := fs.Main() + base + "." + res.FileExt
		fmt.Printf("moving to %q...\n", to)
		err = os.Rename(res.FileName, to)
		if err != nil {
			fmt.Printf("err renaming: %v\n", err)
			return
		}
	}
}
