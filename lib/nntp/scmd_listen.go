package nntp

import (
	"sync"
	"time"
)

type nntpListenState int

const (
	// XLISTENed, no event yet
	nntpListenNone nntpListenState = iota
	// XWAITed, spawned worker
	nntpListenWaiting
	// worker was tasked to send awake, ->None once sent
	nntpListenAwaking
	// worker timed out, sending timeout msg, ->None once sent
	nntpListenTimeout
)

type nntpListenObj struct {
	c *ConnState

	mu sync.Mutex
	cv sync.Cond

	wstate nntpListenState
	nawake bool // incase event occured while wasn't waiting
	werr   bool // did worker erred?
}

func (o *nntpListenObj) cancel() (die bool) {
	docancel := false
	o.mu.Lock()
	if o.wstate == nntpListenWaiting {
		// cancel worker
		docancel = true
		o.wstate = nntpListenNone
		o.mu.Unlock()
		o.cv.Broadcast()
		o.mu.Lock()
	}
	// wait till its done to avoid state confusion
	for o.wstate != nntpListenNone {
		o.cv.Wait()
	}
	die = o.werr
	o.mu.Unlock()

	if docancel {
		AbortOnErr(o.c.w.ResXWaitCancel())
	}

	return
}

func (o *nntpListenObj) awake() (nomore bool) {
	o.mu.Lock()
	if o.wstate == nntpListenWaiting {
		o.wstate = nntpListenAwaking
		o.mu.Unlock()
		o.cv.Broadcast()
	} else {
		// XXX nntpListenAwaking state could be skipped too?
		nomore = o.nawake
		o.nawake = true
		o.mu.Unlock()
	}
	return
}

func (o *nntpListenObj) timeoutfunc() {

	sendtimeout := false

	o.mu.Lock()
	if o.wstate == nntpListenWaiting {
		o.wstate = nntpListenTimeout
		sendtimeout = true
	}
	o.mu.Unlock()

	// don't awake anyone yet as it's not actionable

	if sendtimeout {
		err := o.c.w.ResXWaitTimeout()
		o.mu.Lock()
		o.wstate = nntpListenNone
		o.werr = err != nil
		o.mu.Unlock()
		o.cv.Broadcast()
	}
}

func (o *nntpListenObj) worker(dur int32) {

	var t *time.Timer
	if dur > 0 {
		t = time.AfterFunc(
			time.Duration(dur)*time.Millisecond, o.timeoutfunc)
	}

	o.mu.Lock()
	for o.wstate == nntpListenWaiting {
		o.cv.Wait()
	}
	st := o.wstate
	o.mu.Unlock()

	if t != nil {
		t.Stop()
	}

	if st == nntpListenAwaking {
		err := o.c.w.ResXWaitAwake()
		o.mu.Lock()
		o.wstate = nntpListenNone
		o.werr = err != nil
		o.mu.Unlock()
		o.cv.Broadcast()
	}
}

func cmdXListen(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.prov.SupportsXListen() || true {
		AbortOnErr(c.w.PrintfLine("503 XLISTEN unimplemented"))
		return true
	}
	t := unsafeBytesToStr(args[0])
	if t != "*" {
		AbortOnErr(c.w.PrintfLine("503 XLISTEN %q unimplemented", t))
		return true
	}
	o := c.listen
	if o == nil {
		// allocate new listener
		o = &nntpListenObj{c: c}
		o.cv.L = &o.mu
		// XXX
	}
	return false // XXX
}

func cmdXWait(c *ConnState, args [][]byte, rest []byte) bool {
	// TODO
	AbortOnErr(c.w.PrintfLine("503 XLISTEN unimplemented"))
	return true
}
