package mail

import (
	"bytes"
	"io"
	"sync"

	"nekochan/lib/bufreader"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		// XXX dangerous
		return bufreader.NewBufReaderSize(nil, 4096)
	},
}

var hdrPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func obtainBufReader(r io.Reader) (br *bufreader.BufReader) {
	br = bufPool.Get().(*bufreader.BufReader)
	br.Drop()
	br.ResetErr()
	br.SetReader(r)
	return
}

func dropBufReader(br *bufreader.BufReader) {
	br.SetReader(nil)
	br.ResetErr()
	bufPool.Put(br)
}
