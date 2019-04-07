package nntp

import (
	"crypto/tls"
	tp "net/textproto"

	"centpd/lib/bufreader"
	. "centpd/lib/logx"
)

// sugar because im lazy
type Responder struct {
	*tp.Writer
}

func (r Responder) Abort() {
	panic(ErrAbortHandler)
}

type ConnState struct {
	inbuf [512]byte

	srv     *NNTPServer
	conn    ConnCW
	tlsconn *tls.Conn // TLS connection if activated
	r       *bufreader.BufReader
	dr      *bufreader.DotReader
	w       Responder
	log     Logger

	prov          NNTPProvider
	CurrentGroup  interface{}  // provider-specific
	UserPriv                   // stuff allowed
	authenticated bool         // whether authenticated
	activeLogin   *ActiveLogin // for AUTHINFO USER
}

func (c *ConnState) Cleanup() {
	c.CurrentGroup = nil
}

func (c *ConnState) OpenReader() ArticleReader {
	if c.dr != nil {
		c.dr.Reset()
	} else {
		c.dr = bufreader.NewDotReader(c.r)
	}
	return c.dr
}

func (c *ConnState) pullActiveLogin() (l *ActiveLogin) {
	l = c.activeLogin
	c.activeLogin = nil
	return
}

func (c *ConnState) tlsStarted() bool {
	return c.tlsconn != nil
}

type commandFunc func(c *ConnState, args [][]byte, rest []byte) bool

type command struct {
	cmdfunc    commandFunc
	minargs    int
	maxargs    int
	allowextra bool
	help       string
}
