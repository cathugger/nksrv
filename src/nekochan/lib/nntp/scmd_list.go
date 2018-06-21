package nntp

func listCmdActive(c *ConnState, args [][]byte, rest []byte) bool {
	var wildmat []byte
	if len(args) > 0 {
		wildmat = args[0]
		if !validWildmat(wildmat) {
			c.w.PrintfLine("501 invalid wildmat")
			return true
		}
	}

	if !c.AllowReading {
		c.w.ResAuthRequired()
		return true
	}

	c.w.ResListFollows()
	dw := c.w.DotWriter()
	c.prov.ListActiveGroups(dw, wildmat)
	dw.Close()

	return true
}

func listCmdNewsgroups(c *ConnState, args [][]byte, rest []byte) bool {
	var wildmat []byte
	if len(args) > 0 {
		wildmat = args[0]
		if !validWildmat(wildmat) {
			c.w.PrintfLine("501 invalid wildmat")
			return true
		}
	}

	if !c.AllowReading {
		c.w.ResAuthRequired()
		return true
	}

	c.w.ResListFollows()
	dw := c.w.DotWriter()
	c.prov.ListNewsgroups(dw, wildmat)
	dw.Close()

	return true
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

	if !c.AllowReading {
		c.w.ResAuthRequired()
		return true
	}

	c.w.PrintfLine("282 data follows")
	dw := c.w.DotWriter()
	c.prov.ListNewsgroups(dw, wildmat)
	dw.Close()

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
