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

	dw := c.w.DotWriter()
	c.prov.ListNewsgroups(dw, wildmat)
	dw.Close()

	return true
}

// TODO: need to decide what fields exactly we're going to expose
var overviewFmt = []byte(
	`Subject:
From:
Date:
Message-ID:
References:
:bytes
:lines
`)

func listCmdOverviewFmt(c *ConnState, args [][]byte, rest []byte) bool {
	dw := c.w.DotWriter()
	dw.Write(overviewFmt)
	dw.Close()
	return true
}
