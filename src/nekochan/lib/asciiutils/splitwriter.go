package asciiutils

import "io"

var nl = []byte("\n")

type SplitWriter struct {
	W io.Writer
	N int
	I int
}

func (w *SplitWriter) Write(b []byte) (n int, err error) {
	for {
		if len(b) == 0 {
			return
		}

		if w.I >= w.N {
			_, err = w.W.Write(nl)
			if err != nil {
				return
			}
			w.I = 0
		}

		cb := b
		cw := w.N - w.I
		if len(cb) > cw {
			cb = cb[:cw]
		}
		var x int
		x, err = w.W.Write(cb)
		n += x
		w.I += x
		if err != nil {
			return
		}
		b = b[x:]
	}
}
