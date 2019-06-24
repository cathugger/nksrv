package psqlib

import (
	"io"

	"centpd/lib/cacheengine"
)

func generic2nntpcopyer(
	objid string, objinfo interface{}) (
	msgid CoreMsgIDStr, bpid, gpid postID) {

	x := objinfo.(nntpidinfo)
	return CoreMsgIDStr(objid), x.bpid, x.gpid
}

type nntpCopyer interface {
	cacheengine.CopyDestination

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

func (c *fullNNTPCopyer) CopyFrom(
	src io.Reader, objid string, objinfo interface{}) (
	written int64, err error) {

	msgid, bpid, gpid := generic2nntpcopyer(objid, objinfo)

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

		// successful get - set new id
		if c.gs != nil {
			c.gs.bpid = bpid
			c.gs.gpid = gpid
		}

		c.w.ResArticleFollows(bpid, msgid)
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

func (c *headNNTPCopyer) CopyFrom(
	src io.Reader, objid string, objinfo interface{}) (
	written int64, err error) {

	msgid, bpid, gpid := generic2nntpcopyer(objid, objinfo)

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

		// successful get - set new id
		if c.gs != nil {
			c.gs.bpid = bpid
			c.gs.gpid = gpid
		}

		c.w.ResHeadFollows(bpid, msgid)
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

func (c *bodyNNTPCopyer) CopyFrom(
	src io.Reader, objid string, objinfo interface{}) (
	written int64, err error) {

	msgid, bpid, gpid := generic2nntpcopyer(objid, objinfo)

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

		// successful get - set new id
		if c.gs != nil {
			c.gs.bpid = bpid
			c.gs.gpid = gpid
		}

		c.w.ResBodyFollows(bpid, msgid)
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

func (c statNNTPCopyer) CopyFrom(
	src io.Reader, objid string, objinfo interface{}) (
	written int64, err error) {

	msgid, bpid, gpid := generic2nntpcopyer(objid, objinfo)

	// successful get - set new id
	if c.gs != nil {
		c.gs.bpid = bpid
		c.gs.gpid = gpid
	}

	// interface abuse
	c.w.ResArticleFound(bpid, msgid)
	return 0, nil
}

func (statNNTPCopyer) IsClosed() bool {
	return true
}
