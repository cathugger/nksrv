package mail

import (
	"bytes"
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
