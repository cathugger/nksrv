package main

import (
	"fmt"
	"image"
	"image/gif"
	"io/ioutil"
	"os"

	"nksrv/lib/utils/xface"
)

const usage = `Usage:
	%s imgtoxface [filename.gif] [filename.xface]
	%s xfacetoimg "x-face-content" [filename.gif]
`

func usagequit() {
	fmt.Fprintf(os.Stderr, usage, os.Args[0], os.Args[0])
	os.Exit(1)
}

func xfacetoimg(args []string) {
	var in string
	if len(args) != 0 && args[0] != "" {
		in = args[0]
	} else {
		inb, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
			os.Exit(1)
		}
		in = string(inb)
	}

	img, err := xface.XFaceStringToImg(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decoding x-face: %v\n", err)
		os.Exit(1)
	}

	var out *os.File
	if len(args) < 2 {
		out = os.Stdout
	} else {
		out, err = os.Open(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening output file: %v\n", err)
			os.Exit(1)
		}
	}

	err = gif.Encode(out, img, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding gif: %v\n", err)
		os.Exit(1)
	}

	err = out.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error closing gif: %v\n", err)
		os.Exit(1)
	}
}

func imgtoxface(args []string) {
	var err error

	var in *os.File
	if len(args) != 0 && args[0] != "" {
		in, err = os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening input file: %v\n", err)
			os.Exit(1)
		}
		defer in.Close()
	} else {
		in = os.Stdin
	}

	img, _, err := image.Decode(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decoding input file: %v\n", err)
		return
	}

	s, err := xface.XFaceImgToString(img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error converting image: %v\n", err)
		return
	}

	var out *os.File
	if len(args) < 2 {
		out = os.Stdout
	} else {
		out, err = os.Open(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening output file: %v\n", err)
			return
		}
	}

	_, err = fmt.Fprintf(out, "%s\n", s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing string: %v\n", err)
		return
	}

	err = out.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error closing file: %v\n", err)
		return
	}
}

func main() {
	if len(os.Args) < 2 {
		usagequit()
	}

	switch os.Args[1] {
	case "imgtoxface":
		imgtoxface(os.Args[2:])

	case "xfacetoimg":
		xfacetoimg(os.Args[2:])

	default:
		usagequit()
	}
}
