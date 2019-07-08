package nntp

import "io"

type listCmdListOpener struct {
	Responder
}

func (o listCmdListOpener) OpenDotWriter() (_ io.WriteCloser, err error) {
	err = o.Responder.ResListFollows()
	if err != nil {
		return
	}
	return o.Responder.DotWriter(), nil
}

func (o listCmdListOpener) GetResponder() Responder {
	return o.Responder
}

func listCmdActive(c *ConnState, args [][]byte, rest []byte) bool {
	var wildmat []byte
	if len(args) != 0 {
		wildmat = args[0]
		if !validWildmat(wildmat) {
			AbortOnErr(c.w.PrintfLine("501 invalid wildmat"))
			return true
		}
	}

	if !c.AllowReading && !c.AllowPosting {
		AbortOnErr(c.w.ResAuthRequired())
		return true
	}

	c.prov.ListActiveGroups(listCmdListOpener{c.w}, wildmat)

	return true
}

func listCmdNewsgroups(c *ConnState, args [][]byte, rest []byte) bool {
	var wildmat []byte
	if len(args) > 0 {
		wildmat = args[0]
		if !validWildmat(wildmat) {
			AbortOnErr(c.w.PrintfLine("501 invalid wildmat"))
			return true
		}
	}

	if !c.AllowReading && !c.AllowPosting {
		AbortOnErr(c.w.ResAuthRequired())
		return true
	}

	c.prov.ListNewsgroups(listCmdListOpener{c.w}, wildmat)

	return true
}

type cmdXGTitleOpener struct {
	Responder
}

func (o cmdXGTitleOpener) OpenDotWriter() (_ io.WriteCloser, err error) {
	err = o.Responder.PrintfLine("282 data follows")
	if err != nil {
		return
	}
	return o.Responder.DotWriter(), nil
}

func (o cmdXGTitleOpener) GetResponder() Responder {
	return o.Responder
}

// same as LIST NEWSGROUPS just with different return codes
func cmdXGTitle(c *ConnState, args [][]byte, rest []byte) bool {
	var wildmat []byte
	if len(args) > 0 {
		wildmat = args[0]
		if !validWildmat(wildmat) {
			c.w.PrintfLine("501 invalid wildmat")
			return true
		}
	}

	if !c.AllowReading && !c.AllowPosting {
		c.w.ResAuthRequired()
		return true
	}

	c.prov.ListNewsgroups(cmdXGTitleOpener{c.w}, wildmat)

	return true
}

// TODO: need to decide what fields exactly we're going to expose
// {RFC 2980} Many newsreaders work better if Xref: is one of the optional fields.
var overviewFmt = []byte(
	`Subject:
From:
Date:
Message-ID:
References:
:bytes
:lines
Xref:full
`)

func listCmdOverviewFmt(c *ConnState, args [][]byte, rest []byte) bool {
	c.w.ResListFollows()
	dw := c.w.DotWriter()
	dw.Write(overviewFmt)
	dw.Close()
	return true
}
