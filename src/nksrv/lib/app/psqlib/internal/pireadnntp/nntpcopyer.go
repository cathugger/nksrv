package pireadnntp

import (
	"io"

	"nksrv/lib/cacheengine"
)

func generic2nntpcopyer(
	objid string, objinfo interface{}) (
	msgid TCoreMsgIDStr, bpid, gpid postID) {

	x := objinfo.(nntpidinfo)
	return TCoreMsgIDStr(objid), x.bpid, x.gpid
}

type nntpCopyer interface {
	cacheengine.CopyDestination

	SetGroupState(gs *groupState)
	IsClosed() bool
}

// full
type FullNNTPCopyer struct {
	w  Responder
	dw io.WriteCloser
	gs *groupState
}

func (c *FullNNTPCopyer) SetGroupState(gs *groupState) {
	c.gs = gs
}

func (c *FullNNTPCopyer) CopyFrom(
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

		err = c.w.ResArticleFollows(bpid, msgid)
		if err != nil {
			return
		}
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

func (c *FullNNTPCopyer) IsClosed() bool {
	return c.dw == nil
}

// head
type HeadNNTPCopyer struct {
	w  Responder
	dw io.WriteCloser
	gs *groupState
	st int
}

func (c *HeadNNTPCopyer) SetGroupState(gs *groupState) {
	c.gs = gs
}

func (c *HeadNNTPCopyer) CopyFrom(
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

		err = c.w.ResHeadFollows(bpid, msgid)
		if err != nil {
			return
		}
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

func (c *HeadNNTPCopyer) IsClosed() bool {
	return c.dw == nil
}

// body
type BodyNNTPCopyer struct {
	w  Responder
	dw io.WriteCloser
	gs *groupState
	st int
}

func (c *BodyNNTPCopyer) SetGroupState(gs *groupState) {
	c.gs = gs
}

func (c *BodyNNTPCopyer) CopyFrom(
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

		err = c.w.ResBodyFollows(bpid, msgid)
		if err != nil {
			return
		}
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

func (c *BodyNNTPCopyer) IsClosed() bool {
	return c.dw == nil
}

// stat
type StatNNTPCopyer struct {
	w  Responder
	gs *groupState
}

func (c *StatNNTPCopyer) SetGroupState(gs *groupState) {
	c.gs = gs
}

func (c *StatNNTPCopyer) CopyFrom(
	src io.Reader, objid string, objinfo interface{}) (
	written int64, err error) {

	msgid, bpid, gpid := generic2nntpcopyer(objid, objinfo)

	// successful get - set new id
	if c.gs != nil {
		c.gs.bpid = bpid
		c.gs.gpid = gpid
	}

	// interface abuse
	err = c.w.ResArticleFound(bpid, msgid)
	if err != nil {
		return
	}
	return 0, nil
}

func (*StatNNTPCopyer) IsClosed() bool {
	return true
}
