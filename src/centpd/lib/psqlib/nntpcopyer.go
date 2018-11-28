package psqlib

import (
	"io"
)

type nntpCopyer interface {
	Copy(num uint64, msgid CoreMsgIDStr, src io.Reader) (
		written int64, err error)
	SetGroupState(gs *groupState)
	IsClosed() bool
}

// full
type fullNNTPCopyer struct {
	w  Responder
	dw io.WriteCloser
	gs *groupState
}

func (c *fullNNTPCopyer) SetGroupState(gs *groupState) {
	c.gs = gs
}

func (c *fullNNTPCopyer) Copy(
	num uint64, msgid CoreMsgIDStr, src io.Reader) (
	written int64, err error) {

	buf := make([]byte, 32*1024)

	var nr, nw int
	var er, ew error

	// initialise if we successfuly read something
	if c.dw == nil {
		nr, er = src.Read(buf)
		if nr <= 0 && er != nil {
			// abort on error before initialisation if no data - avoid begining empty incomplete message
			err = er
			return
		}
		if c.gs != nil {
			c.gs.pid = num
		}
		c.w.ResArticleFollows(num, msgid)
		c.dw = c.w.DotWriter()
	}

	for {
		if nr > 0 {
			nw, ew = c.dw.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				// read EOF isn't error :>
				err = er
			}
			break
		}
		nr, er = src.Read(buf)
	}

	if err == nil {
		err = c.dw.Close()
		c.dw = nil
	}

	return
}

func (c *fullNNTPCopyer) IsClosed() bool {
	return c.dw == nil
}

// head
type headNNTPCopyer struct {
	w  Responder
	dw io.WriteCloser
	gs *groupState
	st int
}

func (c *headNNTPCopyer) SetGroupState(gs *groupState) {
	c.gs = gs
}

func (c *headNNTPCopyer) Copy(num uint64, msgid CoreMsgIDStr, src io.Reader) (written int64, err error) {
	buf := make([]byte, 32*1024)

	var nr, nw int
	var er, ew error

	// initialise if we successfuly read something
	if c.dw == nil {
		nr, er = src.Read(buf)
		if nr <= 0 && er != nil {
			// abort on error before initialisation if no data - avoid begining empty incomplete message
			err = er
			return
		}
		if c.gs != nil {
			c.gs.pid = num
		}
		c.w.ResHeadFollows(num, msgid)
		c.dw = c.w.DotWriter()
	}

	const (
		sNL = iota
		sNonNL
		sEOF
	)

	for {
		if c.st == sEOF {
			nr = 0
			er = io.EOF
		}
		for i := 0; i < nr; i++ {
			if buf[i] != '\n' {
				if c.st == sNL {
					c.st = sNonNL
				}
			} else {
				if c.st == sNonNL {
					c.st = sNL
				} else {
					c.st = sEOF
					er = io.EOF
					nr = i
					break
				}
			}
		}
		if nr > 0 {
			nw, ew = c.dw.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er == io.EOF && c.st != sEOF {
				er = io.ErrUnexpectedEOF
			}
			if er != io.EOF {
				// read EOF isn't error :>
				err = er
			}
			break
		}
		nr, er = src.Read(buf)
	}

	if err == nil {
		err = c.dw.Close()
		c.dw = nil
	}

	return
}

func (c *headNNTPCopyer) IsClosed() bool {
	return c.dw == nil
}

// body
type bodyNNTPCopyer struct {
	w  Responder
	dw io.WriteCloser
	gs *groupState
	st int
}

func (c *bodyNNTPCopyer) SetGroupState(gs *groupState) {
	c.gs = gs
}

func (c *bodyNNTPCopyer) Copy(
	num uint64, msgid CoreMsgIDStr, src io.Reader) (
	written int64, err error) {

	buf := make([]byte, 32*1024)

	var nr, nw int
	var er, ew error
	var sr int

	// initialise if we successfuly read something
	if c.dw == nil {
		nr, er = src.Read(buf)
		if nr <= 0 && er != nil {
			// abort on error before initialisation if no data - avoid begining empty incomplete message
			err = er
			return
		}
		if c.gs != nil {
			c.gs.pid = num
		}
		c.w.ResBodyFollows(num, msgid)
		c.dw = c.w.DotWriter()
	}

	const (
		sNL = iota
		sNonNL
		sBody
	)

	for {
		for sr = 0; c.st != sBody && sr < nr; sr++ {
			if buf[sr] != '\n' {
				c.st = sNonNL
			} else {
				if c.st == sNonNL {
					c.st = sNL
				} else {
					c.st = sBody
				}
			}
		}
		if nr > sr {
			nw, ew = c.dw.Write(buf[sr:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr-sr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er == io.EOF && c.st != sBody {
				er = io.ErrUnexpectedEOF
			}
			if er != io.EOF {
				// read EOF isn't error :>
				err = er
			}
			break
		}
		nr, er = src.Read(buf)
	}

	if err == nil {
		err = c.dw.Close()
		c.dw = nil
	}

	return
}

func (c *bodyNNTPCopyer) IsClosed() bool {
	return c.dw == nil
}

// stat
type statNNTPCopyer struct {
	w  Responder
	gs *groupState
}

func (c *statNNTPCopyer) SetGroupState(gs *groupState) {
	c.gs = gs
}

func (c statNNTPCopyer) Copy(
	num uint64, msgid CoreMsgIDStr, src io.Reader) (
	written int64, err error) {

	if c.gs != nil {
		c.gs.pid = num
	}
	// interface abuse
	c.w.ResArticleFound(num, msgid)
	return 0, nil
}

func (statNNTPCopyer) IsClosed() bool {
	return true
}
