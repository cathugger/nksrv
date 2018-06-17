package nntp

import (
	"bufio"
	"fmt"
	"net"
	tp "net/textproto"
	"sync"
	"time"

	"nekochan/lib/bufreader"
	. "nekochan/lib/logx"
)

// sugar because im lazy
type Responder struct {
	*tp.Writer
}

type ConnState struct {
	srv  *NNTPServer
	conn ConnCW
	r    *bufreader.BufReader
	w    Responder

	prov         NNTPProvider
	CurrentGroup interface{}
}

func parseKeyword(b []byte) int {
	i := 0
	l := len(b)
	for {
		if i >= l {
			return i
		}
		c := b[i]
		if c == ' ' || c == '\t' {
			return i
		}
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
		i++
	}
}

func cmdVoid(c *ConnState, args [][]byte, rest []byte) bool {
	if len(rest) != 0 {
		c.w.PrintfLine("501 command must not start with space")
	}
	// otherwise ignore
	return true
}

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

func NewNNTPServer(logx LoggerX) *NNTPServer {
	s := &NNTPServer{}
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

func (s *NNTPServer) handleConnection(c ConnCW) {
	r := bufreader.NewBufReader(c)
	cs := &ConnState{
		srv:  s,
		conn: c,
		r:    r,
		w:    Responder{Writer: tp.NewWriter(bufio.NewWriter(c))},
	}

	graceful := cs.serve()

	if graceful && c.CloseWrite() == nil {
		r.Skip(1) // ignore return, it's error to send anything after quit command
	}

	c.Close()
	s.unregisterConn(c)
}

func (s *NNTPServer) ListenAndServe(network, addr string, listenParam ListenParam) error {
	raddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		s.log.LogPrintf(ERROR, "failed to resolve {%s}%s: %v", network, addr, err)
		return err
	}
	s.log.LogPrintf(INFO, "{%s}%s resolved to %s", network, addr, raddr)

	tl, err := net.ListenTCP(network, raddr)
	if err != nil {
		s.log.LogPrintf(ERROR, "failed to listen on {%s}%s: %v", network, raddr, err)
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
				s.log.LogPrintf(ERROR, "accept error: %v; retrying in %v", err, delay)
				time.Sleep(delay)
				continue
			}
			s.log.LogPrintf(ERROR, "accept error: %v; aborting", err)
			return err
		}
		s.log.LogPrintf(NOTICE, "accepted %s on %s", c.RemoteAddr(), c.LocalAddr())
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

func cmdList(c *ConnState, args [][]byte, rest []byte) bool {
	args = args[:0] // reuse

	if len(rest) == 0 {
		listCmdActive(c, args, nil)
		return true
	}

	x := parseKeyword(rest)

	cmd, ok := listCommandMap[string(rest[:x])]
	if !ok {
		c.w.PrintfLine("501 unrecognised LIST keyword")
		return true
	}

	if x >= len(rest) {
		goto argsparsed
	}

	for {
		// skip spaces
		for {
			x++
			if x >= len(rest) {
				goto argsparsed
			}
			if rest[x] != ' ' && rest[x] != '\t' {
				break
			}
		}
		if len(args) >= cmd.maxargs {
			if !cmd.allowextra {
				c.w.PrintfLine("501 too much parameters")
			} else {
				cmd.cmdfunc(c, args, rest[x:])
			}
			return true
		}
		sx := x
		// skip non-spaces
		for {
			x++
			if x >= len(rest) {
				args = append(args, rest[sx:x])
				goto argsparsed
			}
			if rest[x] == ' ' || rest[x] == '\t' {
				args = append(args, rest[sx:x])
				break
			}
		}
	}
argsparsed:
	if len(args) < cmd.minargs {
		c.w.PrintfLine("501 not enough parameters")
		return true
	}
	cmd.cmdfunc(c, args, nil)
	return true
}

func (c *ConnState) serve() bool {
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
					// generic error while draining
					return false
				}
				c.w.PrintfLine("501 command too long")
				continue
			} else {
				// generic read error, just quit as socket prolly broke
				return false
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

		x := parseKeyword(incmd)
		cmd, ok := commandMap[string(incmd[:x])]
		if !ok {
			c.w.PrintfLine("500 unrecognised command")
			continue
		}

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
				} else {
					if !cmd.cmdfunc(c, args, incmd[x:]) {
						return true
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
			goto nextcommand
		}
		if !cmd.cmdfunc(c, args, nil) {
			return true
		}
	nextcommand:
	}
}