package nntp

import . "nksrv/lib/utils/logx"

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

type NNTPPusher struct {
	NNTPClient

	// XXX
}

type articleNotif struct {
	cmsgid string
	glpid  uint64
	ginfo  []struct {
		group string
		grpid uint64
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

/*
 * planned pipeline:
 * notify func gets called
 * it sends thru 0-sized channel
 * nonBlockArticlePumper receives and sends thru buffered channel
 * sends to worker
 *
 * worker:
 * initially, checks what groups server can recv
 * clears unsendable-to groups from trackdb
 * adds any new sendable-to groups to trackdb,
 *   initialized to either 0 or {current_max_gpost_num} depending on setting
 * checks for each group if {current_max_gpost_num} > max_gpost_num in trackdb
 * for each such article gets msgid and groups off db, and offers to server
 * for each sent/server didn't need conclusion, writes to db
 * listens on channel and offers to server
 * after every sent/rejected, writes new stats to db
 * if event was nil, does initial check procedure
 *
 * stat writer:
 * should queue and merge writes
 * should disallow write if trackdb is increased more than by one
 * unless there's few pending things
 * or expected tracked ver > actual db tracked ver
 * update algo:
 *   if db_ver >= expected_ver, db_ver := max(new_ver,db_ver)
 * this can fail update if theres mismatch...
 *   but that can be detected, and write held up
 *   until some thread gets notified about gap and fixes it up
 *   note that only single thread should do gap fixing up
 *   that should be guarded by lock
 *
 * determining what groups are sendable-to:
 * we should list current server's groups
 * and also match these with autoadd-wildcards
 * autoadd-wildcards would be obtained by extension
 * or otherwise could be statically set in config
 * static config is sorta ass tbh
 * either way, these would be matched against current grouplist of ours
 * this sendable-to list would need to be periodically rechecked
 * period should be configurable and default to reasonable value (15 mins?)
 *
 * server accept policy:
 * server may either accept only to currently available groups
 * or it may use wildcard expressions to match groups it accepts
 * if server don't specify a way, then whether server can autoaccept
 *   and what it can autoaccept should be configurable
 */

func (c *NNTPPusher) sendArticleToServer(
	cmsgid string, glpid uint64) (error, bool) {
	// TODO
	return nil, false
}

func (c *NNTPPusher) pushworker(
	inev <-chan articleNotif) (err error, fatal bool) {

	// x
	var ev articleNotif

	offer := func() {
		err = c.w.PrintfLine("IHAVE <%s>", ev.cmsgid)
		if err != nil {
			// XXX
			panic(err)
		}
		var code uint
		code, _, err, fatal = c.readResponse()
		if err != nil {
			c.log.LogPrintf(DEBUG, "readResponse() err: %v", err)
			return
		}
		if code == 335 {
			// server wants
			err, fatal = c.sendArticleToServer(ev.cmsgid, ev.glpid)
			if err != nil {
				return
			}
			code, _, err, fatal = c.readResponse()
			if err != nil {
				c.log.LogPrintf(DEBUG, "readResponse() err: %v", err)
				return
			}
		} else if code == 435 {
			// server doesn't want
		} else {
			// some error
		}
	}
	_ = offer

	for {
		if ev.cmsgid != "" {
			// valid event, just try pushing this one
		} else {
			// gap, figure out how much we need to send
		}
	}
}
