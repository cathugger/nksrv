package nntp

import (
	"bufio"
	"fmt"
	"io"
	"net"
	tp "net/textproto"
	"sync"
	"time"

	"nekochan/lib/bufreader"
	. "nekochan/lib/logx"
)

// net.Conn with additional CloseWrite() function
type ConnCW interface {
	net.Conn
	CloseWrite() error
}

// similar to net.Listener except with AcceptCW() function
type ListenerCW interface {
	AcceptCW() (ConnCW, error)
	Close() error
	Addr() net.Addr
}

type NNTPServer struct {
	log  Logger
	logx LoggerX
	prov NNTPProvider

	mu          sync.Mutex
	closing     bool
	wg          sync.WaitGroup
	listeners   map[ListenerCW]struct{}
	connections map[ConnCW]struct{}
}

type ListenParam struct {
	KeepAlive time.Duration
}

// used to set up connection properties
type tcpListenerWrapper struct {
	*net.TCPListener
	keepAlive time.Duration
}

var _ ListenerCW = tcpListenerWrapper{}

func (w tcpListenerWrapper) AcceptCW() (ConnCW, error) {
	c, err := w.AcceptTCP()
	if err == nil {
		c.SetLinger(0)
		if w.keepAlive != 0 {
			c.SetKeepAlive(true)
			c.SetKeepAlivePeriod(w.keepAlive)
		} else {
			c.SetKeepAlive(false)
		}
	}
	// XXX incase c == nil, returns not nil interface but interface which points to nil
	return c, err
}

func NewNNTPServer(prov NNTPProvider, logx LoggerX) *NNTPServer {
	s := &NNTPServer{
		prov: prov,
		logx: logx,
	}
	s.log = NewLogToX(logx, fmt.Sprintf("nntpsrv.%p", s))
	return s
}

func (s *NNTPServer) tryRegister(l ListenerCW) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closing {
		return false, nil // XXX better err code
	}
	if s.listeners == nil {
		s.listeners = make(map[ListenerCW]struct{})
	}
	if _, ok := s.listeners[l]; ok {
		// already listening
		return false, nil // XXX better err code
	}
	s.listeners[l] = struct{}{}
	s.wg.Add(1)
	return true, nil
}

func (s *NNTPServer) checkClosing(l ListenerCW) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closing {
		return false
	}
	delete(s.listeners, l)
	return true
}

func (s *NNTPServer) registerConn(c ConnCW) {
	s.mu.Lock()
	if s.connections == nil {
		s.connections = make(map[ConnCW]struct{})
	}
	s.connections[c] = struct{}{}
	s.mu.Unlock()
}

func (s *NNTPServer) unregisterConn(c ConnCW) {
	s.mu.Lock()
	if s.connections != nil {
		delete(s.connections, c)
	}
	s.mu.Unlock()
}

const (
	cGraceful = iota
	cHangup
	cError
)

func (s *NNTPServer) handleConnection(c ConnCW) {
	r := bufreader.NewBufReader(c)
	cs := &ConnState{
		srv:  s,
		conn: c,
		r:    r,
		prov: s.prov,
		w:    Responder{tp.NewWriter(bufio.NewWriter(c)), c},
	}
	cs.log = NewLogToX(
		s.logx, fmt.Sprintf("nntpsrv.%p.client.%p-%s", s, cs, c.RemoteAddr()))
	s.setupClientDefaults(cs)

	if cs.AllowPosting {
		cs.w.PrintfLine("200 hello! posting allowed.")
	} else {
		cs.w.PrintfLine("201 hello! posting forbidden.")
	}

	reason := cs.serveClient()

	// XXX incase cHangup, we have no way to check if other side ACK'd data before killing socket.
	if reason != cError && c.CloseWrite() == nil && reason == cGraceful {
		r.Discard(1) // ignore return, it's error to send anything after quit command
	}

	c.Close()
	s.unregisterConn(c)
}

func (s *NNTPServer) ListenAndServe(
	network, addr string, listenParam ListenParam) error {

	raddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		s.log.LogPrintf(ERROR, "failed to resolve {%s}%s: %v", network, addr, err)
		return err
	}
	s.log.LogPrintf(INFO, "{%s}%s resolved to %s", network, addr, raddr)

	tl, err := net.ListenTCP(network, raddr)
	if err != nil {
		s.log.LogPrintf(ERROR,
			"failed to listen on {%s}%s: %v", network, raddr, err)
		return err
	}
	s.log.LogPrintf(INFO, "listening on {%s}%s", network, raddr)

	w := tcpListenerWrapper{
		TCPListener: tl,
		keepAlive:   listenParam.KeepAlive,
	}

	return s.Serve(w)
}

func (s *NNTPServer) Serve(l ListenerCW) error {
	if ok, err := s.tryRegister(l); !ok {
		return err
	}
	defer s.wg.Done()

	s.log.LogPrintf(INFO, "accepting connections on %s", l.Addr())

	delay := time.Duration(0)
	for {
		c, err := l.AcceptCW()
		if err != nil {
			if s.checkClosing(l) {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else {
					delay *= 2
				}
				if max := 1 * time.Second; delay > max {
					delay = max
				}
				s.log.LogPrintf(
					ERROR, "accept error: %v; retrying in %v", err, delay)
				time.Sleep(delay)
				continue
			}
			s.log.LogPrintf(ERROR, "accept error: %v; aborting", err)
			return err
		}
		s.log.LogPrintf(
			NOTICE, "accepted %s on %s", c.RemoteAddr(), c.LocalAddr())
		// track it, we gonna need it when closing, as Serve() functions may prematurely return and thats OK
		s.registerConn(c)
		// spawn handler
		go s.handleConnection(c)
	}
}

func (s *NNTPServer) Close() bool {
	s.mu.Lock()

	if s.closing {
		s.mu.Unlock()
		return false
	}
	s.closing = true

	// new listeners wont spawn, but closed ones may deregister
	for l := range s.listeners {
		l.Close()
	}
	// listeners should just die off now

	s.mu.Unlock()

	// wait for all Serve()s to quit
	// they can spawn new active connections even when we signal closing state
	s.wg.Wait()
	// they all should be ded now

	s.mu.Lock()

	// now kill all active connections
	// locked because they sometimes can remove themselves
	for c := range s.connections {
		c.Close()
	}
	// maybe have sort of waitgroup here too?
	// not sure if needed, as sockets are closed at this point anyway

	// we're done closing, so allow new servers to spawn later
	s.closing = false

	s.mu.Unlock()
	// done ^_^
	return true
}

func (c *ConnState) serveClient() int {
	var inbuf [512]byte
	args := make([][]byte, 0)

	for {
		i, e := c.r.ReadUntil(inbuf[:], '\n')
		if e != nil {
			if e == bufreader.ErrDelimNotFound {
				// command line too big to process, drain and signal error
				for {
					_, e = c.r.ReadUntil(inbuf[:], '\n')
					if e != bufreader.ErrDelimNotFound {
						break
					}
				}
				if e != nil {
					if e == io.EOF {
						return cHangup
					} else {
						return cError
					}
				}
				c.w.PrintfLine("501 command too long")
				continue
			} else if e == io.EOF {
				return cHangup
			} else {
				return cError
			}
		}

		var incmd []byte
		if i > 1 && inbuf[i-2] == '\r' {
			incmd = inbuf[:i-2]
		} else {
			incmd = inbuf[:i-1]
		}
		for _, ch := range incmd {
			if ch == '\000' || ch == '\r' {
				c.w.PrintfLine("501 command contains illegal characters")
				continue
			}
		}

		c.log.LogPrintf(DEBUG, "got %q", incmd)

		x := parseKeyword(incmd)
		cmd, ok := commandMap[string(incmd[:x])]
		if !ok {
			c.w.PrintfLine("500 unrecognised command")
			c.log.LogPrintf(WARN, "unrecognised command %q", incmd[:x])
			continue
		}
		c.log.LogPrintf(INFO, "processing command %q", incmd[:x])

		args = args[:0] // reuse

		if x >= len(incmd) {
			goto argsparsed
		}

		for {
			// skip spaces
			for {
				x++
				if x >= len(incmd) {
					goto argsparsed
				}
				if incmd[x] != ' ' && incmd[x] != '\t' {
					break
				}
			}
			if len(args) >= cmd.maxargs {
				if !cmd.allowextra {
					c.w.PrintfLine("501 too much parameters")
					c.log.LogPrintf(WARN, "too much parameters")
				} else {
					if !cmd.cmdfunc(c, args, incmd[x:]) {
						return cGraceful
					}
				}
				goto nextcommand
			}
			sx := x
			// skip non-spaces
			for {
				x++
				if x >= len(incmd) {
					args = append(args, incmd[sx:x])
					goto argsparsed
				}
				if incmd[x] == ' ' || incmd[x] == '\t' {
					args = append(args, incmd[sx:x])
					break
				}
			}
		}
	argsparsed:
		if len(args) < cmd.minargs {
			c.w.PrintfLine("501 not enough parameters")
			c.log.LogPrintf(WARN, "not enough parameters")
			goto nextcommand
		}
		if !cmd.cmdfunc(c, args, nil) {
			return cGraceful
		}
	nextcommand:
	}
}
