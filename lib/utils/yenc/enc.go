package yenc

import "io"

type YEncoder struct {
	buf  [1000]byte
	bufi int
	w    io.Writer
}

const lineLength = 990

func (e *YEncoder) Write(b []byte) (n int, err error) {
	for i, c := range b {
		if e.bufi >= lineLength {
			var nn int
			nn, err = e.w.Write(e.buf[:e.bufi])
			if err != nil {
				if nn != 0 {
					copy(e.buf[:], e.buf[nn:e.bufi])
					e.bufi -= nn
				}
				n = i
				return
			}
			if nn != e.bufi {
				panic("wrong written bytes")
			}
			e.bufi = 0
		}

		c += 42
		if c == '\000' || c == '\r' || c == '\n' || c == '=' ||
			/* pretty much all yEnc uses are inside dotwriter */
			(e.bufi == 0 && c == '.') {

			c += 64
			e.buf[e.bufi] = '='
			e.bufi++
		}
		e.buf[e.bufi] = c
		e.bufi++

		if e.bufi >= lineLength {
			e.buf[e.bufi], e.buf[e.bufi+1] = '\r', '\n'
			e.bufi += 2
		}
	}

	return len(b), nil
}

func (e *YEncoder) Close() (err error) {
	if e.bufi != 0 {
		if e.buf[e.bufi-1] != '\n' {
			e.buf[e.bufi], e.buf[e.bufi+1] = '\r', '\n'
			e.bufi += 2
		}
		var nn int
		nn, err = e.w.Write(e.buf[:e.bufi])
		if err != nil {
			if nn > 0 {
				copy(e.buf[:], e.buf[nn:e.bufi])
				e.bufi -= nn
			}
			return
		}
		if nn != e.bufi {
			panic("wrong written bytes")
		}
		e.bufi = 0
	}
	return
}
