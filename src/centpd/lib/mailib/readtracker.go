package mailib

import "io"

type readTracker struct {
	R       io.Reader
	HasNull bool
	Has8Bit bool
}

func (r *readTracker) Read(b []byte) (n int, err error) {
	n, err = r.R.Read(b)
	for _, c := range b[:n] {
		if c == 0x00 {
			r.HasNull = true
		}
		if c >= 0x80 {
			r.Has8Bit = true
		}
	}
	return
}
