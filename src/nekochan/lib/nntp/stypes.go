package nntp

import (
	"io"
	tp "net/textproto"

	"nekochan/lib/bufreader"
	. "nekochan/lib/logx"
)

// sugar because im lazy
type Responder struct {
	*tp.Writer
	c io.Closer
}

func (r Responder) Abort() {
	panic(ErrAbortHandler)
}

type ConnState struct {
	srv  *NNTPServer
	conn ConnCW
	r    *bufreader.BufReader
	dr   *bufreader.DotReader
	w    Responder
	log  Logger

	prov         NNTPProvider
	CurrentGroup interface{}
	AllowReading bool
	AllowPosting bool
}

func (c *ConnState) OpenReader() ArticleReader {
	if c.dr != nil {
		c.dr.Reset()
	} else {
		c.dr = bufreader.NewDotReader(c.r)
	}
	return c.dr
}

type commandFunc func(c *ConnState, args [][]byte, rest []byte) bool

type command struct {
	cmdfunc    commandFunc
	minargs    int
	maxargs    int
	allowextra bool
	help       string
}
