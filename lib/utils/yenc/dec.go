package yenc

import (
	"errors"
	"io"

	"nksrv/lib/utils/text/bufreader"
)

var errUnterminatedEscape = errors.New("yenc: unterminated escape")

type YDecoder struct {
	br      *bufreader.BufReader
	line    [1000]byte
	linex   int
	linelen int
}

func (d *YDecoder) fetch() (err error) {
	var n int
	n, err = d.br.ReadUntil(d.line[:], '\n')
	if n > 0 && d.line[n-1] == '\n' {
		n--
	}
	if n > 0 && d.line[n-1] == '\r' {
		n--
	}

	if len(d.line) >= 2 && d.line[0] == '=' && d.line[1] == 'y' {
		// skip any line starting =y
		return
	}

	d.linex = 0
	d.linelen = 0
	for i := 0; i < n; i++ {
		c := d.line[i]
		if c == '=' {
			i++
			if i >= n {
				if err == nil {
					err = errUnterminatedEscape
				}
				break
			}
			c = d.line[i] - 64
		}
		d.line[d.linelen] = c - 42
		d.linelen++
	}

	return
}

func (d *YDecoder) Read(p []byte) (n int, err error) {
	for d.linex == d.linelen {
		if err != nil {
			return 0, err
		}
		err = d.fetch()
	}

	n = copy(p, d.line[d.linex:d.linelen])
	d.linex += n
	return
}

func (d *YDecoder) WriteTo(w io.Writer) (n int64, err error) {
	for {
		for d.linex == d.linelen {
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				return 0, err
			}
			err = d.fetch()
		}

		wn, we := w.Write(d.line[d.linex:d.linelen])

		n += int64(wn)
		d.linex += wn

		if we != nil {
			if err == nil || err == io.EOF {
				err = we
			}
			return
		}
	}
}
