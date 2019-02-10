package main

import (
	"fmt"
	"image/gif"
	"os"

	"centpd/lib/xface"
)

func main() {
	if len(os.Args) != 2 && len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s \"x-face-content\" [filename.gif]\n", os.Args[0])
		os.Exit(1)
	}

	img, err := xface.XFaceStringToImg(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading x-face: %v\n", err)
		os.Exit(1)
	}

	var out *os.File
	if len(os.Args) < 3 {
		out = os.Stdout
	} else {
		out, err = os.Open(os.Args[2])
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

	return
}
