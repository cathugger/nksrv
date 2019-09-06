package nntp

/*

type nntpCopyer interface {
	Copy(num uint64, msgid CoreMsgIDStr, src io.Reader) (written int64, err error)
	IsClosed() bool
}

type ArticlePoster struct {
	w  *tp.Writer
	dw io.WriteCloser
}

var _ nntpCopyer = (*ArticlePoster)(nil)

func (c *ArticlePoster) Copy(num uint64, msgid CoreMsgIDStr, src io.Reader) (written int64, err error) {
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

*/

type articleNotif struct {
	cmsgid string
	ginfo  []struct {
		group string
		gpid  uint64
	}
}

func nonBlockArticlePumper(in <-chan articleNotif, out chan<- articleNotif) {
	var p articleNotif
	var n bool
	var ok bool
	for {
		if !n {
			// no pending article, so we have all the time in our hands
			p, ok = <-in
			n = true
		} else {
			// we have something pending so try handling both
			select {
			case out <- p:
				// oh cool we finally got current p out
				n = false
				continue
			case _, ok = <-in:
				// p still contains data, so null it to indicate loss
				p = articleNotif{}
			}
		}
		// chan closed?
		if !ok {
			// we're done there if so
			close(out)
			return
		}
		// try putting out pending article first
		select {
		case out <- p:
			// we sent it out
			n = false
		default:
			// nope, it'd block
		}
	}
}
