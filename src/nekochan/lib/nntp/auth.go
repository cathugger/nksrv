package nntp

func (s *NNTPServer) setupClientDefaults(c *ConnState) {
	// TODO
	c.AllowReading = true
	c.AllowPosting = true
}

func (c *ConnState) advertiseAuth() bool {
	// TODO
	return false
}
