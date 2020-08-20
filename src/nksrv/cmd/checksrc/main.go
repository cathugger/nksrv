package main

import (
	"bytes"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"golang.org/x/crypto/blake2b"

	ht "nksrv/lib/utils/hashtools"
)

func printUsage(f io.Writer) {
	fmt.Fprintf(f, "Usage: %s file/dir filename/dirname\n", os.Args[0])
}

func printFile(n string) {
	fmt.Printf("%q: ", n)
}

func printResult(s string) {
	fmt.Printf("%s\n", s)
}

func printReadErr(e error) {
	fmt.Print("read error: %v\n", e)
}

func printIDK(n string) {
	printFile(n)
	printResult("idk")
}

func processfile(name string) {
	// cut extension
	id := name
	if i := strings.IndexByte(id, '.'); i >= 0 {
		id = id[:i]
	}

	ourfmt := strings.HasSuffix(id, "-b2b")
	if !ourfmt {
		printIDK(name)
		return
	}

	id = id[:len(id)-len("-b2b")]

	dl, err := ht.LowerBase32Enc.DecodeString(id)
	if err != nil || len(dl) < 28 {
		printIDK(name)
		return
	}

	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed opening %q: %v", name, err)
		return
	}
	defer f.Close()

	var h hash.Hash
	switch len(dl) {
	case 28:
		h, _ = blake2b.New(28, nil)
	case 32:
		h, _ = blake2b.New(32, nil)
	default:
		printIDK(name)
		return
	}

	printFile(name)
	_, err = io.Copy(h, f)
	if err != nil {
		printReadErr(err)
		return
	}
	got := h.Sum(nil)
	if bytes.Equal(dl, got) {
		printResult("okay")
	} else {
		printResult("wrong")
	}
}

func main() {
	if len(os.Args) != 3 {
		printUsage(os.Stderr)
		os.Exit(1)
	}

	arg := os.Args[2]

	switch os.Args[1] {
	case "file":
		dname := path.Dir(arg)
		err := os.Chdir(dname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on chdir into %q: %v", dname, err)
			os.Exit(1)
		}
		processfile(path.Base(arg))
	case "dir":
		err := os.Chdir(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on chdir into %q: %v", arg, err)
			os.Exit(1)
		}
		fis, err := ioutil.ReadDir(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading dir %q: %v", arg, err)
			os.Exit(1)
		}
		for _, f := range fis {
			n := f.Name()
			if n != "" && n[0] != '.' && n[0] != '_' && !f.IsDir() {
				processfile(n)
			}
		}
	default:
		printUsage(os.Stderr)
		os.Exit(1)
	}
}
