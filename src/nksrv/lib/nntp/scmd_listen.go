package nntp

func cmdXListen(c *ConnState, args [][]byte, rest []byte) bool {
	// we expect args to be '*' and can't handle any other value, so just ignore
	// also ignore any extra args
	// TODO
	AbortOnErr(c.w.PrintfLine("503 XLISTEN unimplemented"))
	return true
}

func cmdXWait(c *ConnState, args [][]byte, rest []byte) bool {
	// TODO
	AbortOnErr(c.w.PrintfLine("503 XLISTEN unimplemented"))
	return true
}
