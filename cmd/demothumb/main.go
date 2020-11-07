package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"nksrv/lib/app/demo/demohelper"
	"nksrv/lib/utils/emime"
	fl "nksrv/lib/utils/logx/filelogger"
	"nksrv/lib/utils/fs/fstore"
	. "nksrv/lib/utils/logx"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/thumbnailer/extthm"
	"nksrv/lib/thumbnailer/gothm"
)

func doFile(
	arg string,
	fs *fstore.FStore,
	thm thumbnailer.Thumbnailer, tcfg thumbnailer.ThumbConfig) (
	ok bool) {

	f, err := os.Open(arg)
	if err != nil {
		fmt.Printf("err opening %q: %v\n", arg, err)
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return
	}
	fsize := st.Size()

	base := path.Base(arg)
	var ext string
	if i := strings.LastIndexByte(base, '.'); i >= 0 {
		ext = base[i+1:]
	}

	mtype := emime.MIMETypeByExtension(ext)

	fmt.Printf("processing %q (ext %q mime %q)...\n", arg, ext, mtype)

	res, err := thm.ThumbProcess(f, ext, mtype, fsize, tcfg)
	if err != nil {
		fmt.Printf("err thumbnailing: %v", err)
		return
	}
	if res.DBSuffix == "" {
		fmt.Printf("thumbnailer didn't work for this file\n")
		return true
	}

	fmt.Printf("thumbnailed %q, res %#v\n", base, res)

	to := fs.Main() + base + "." + res.CF.Suffix
	fmt.Printf("moving to %q...\n", to)
	err = os.Rename(res.CF.FullTmpName, to)
	if err != nil {
		fmt.Printf("err renaming: %v\n", err)
		return
	}

	for i := range res.CE {
		to = fs.Main() + base + "." + res.CE[i].Suffix
		fmt.Printf("moving to %q...\n", to)
		err = os.Rename(res.CE[i].FullTmpName, to)
		if err != nil {
			fmt.Printf("err renaming: %v\n", err)
			return
		}
	}

	return true
}

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

	fs, err := fstore.OpenFStore(fstore.Config{
		Path:    *thumbdir,
		Private: "demothumb",
	})
	if err != nil {
		fmt.Printf("err opening fstore: %v\n", err)
		return
	}
	fs.DeclareDir("tmp", false)

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
		if !doFile(arg, &fs, thm, tcfg) {
			return
		}
	}
}
