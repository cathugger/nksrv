package main

import (
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/crypto/blake2b"

	ht "nksrv/lib/hashtools"
)

func printUsage(f io.Writer) {
	fmt.Fprintf(f,
		"Usage: %s dir algo_id\n"+
			"Algos: b2b-224-base32 b2b-256-base32\n",
		os.Args[0])
}

func printFile(n string) {
	fmt.Printf("%q: ", n)
}

func printResult(s string) {
	fmt.Printf("%q\n", s)
}

func printReadErr(e error) {
	fmt.Print("read error: %v\n", e)
}

func processfile(h hash.Hash, name string) {

	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed opening %q: %v", name, err)
		return
	}
	defer f.Close()

	ext := ""
	if i := strings.LastIndexByte(name, '.'); i >= 0 && i < len(name)-1 {
		ext = name[i:]
	}

	h.Reset()

	printFile(name)
	_, err = io.Copy(h, f)
	if err != nil {
		printReadErr(err)
		return
	}
	res := h.Sum(nil)
	resstr :=
		ht.LowerBase32Enc.EncodeToString(res) +
			"-b2b" + ext
	printResult(resstr)
}

func main() {
	if len(os.Args) != 3 {
		printUsage(os.Stderr)
		os.Exit(1)
	}
	dname := os.Args[1]
	err := os.Chdir(dname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error on chdir into %q: %v", dname, err)
		os.Exit(1)
	}
	algostr := os.Args[2]
	var h hash.Hash
	switch algostr {
	case "b2b-224-base32":
		h, _ = blake2b.New(28, nil)
	case "b2b-256-base32":
		h, _ = blake2b.New(32, nil)
	default:
		printUsage(os.Stderr)
		os.Exit(1)
	}
	fis, err := ioutil.ReadDir(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading dir %q: %v", dname, err)
		os.Exit(1)
	}
	for _, f := range fis {
		n := f.Name()
		if n != "" && n[0] != '.' && n[0] != '_' && !f.IsDir() {
			processfile(h, n)
		}
	}
}
