package cachepub

import (
	"io"
	"sync"
)

// cached publisher

type readerAtWriter interface {
	io.ReaderAt
	io.Writer
}

type CachePub struct {
	f readerAtWriter

	mu   sync.RWMutex
	cond *sync.Cond
	n    int64 // how much bytes we wrote
	err  error // io.EOF on finish or error
}

type Reader struct {
	c *CachePub
	n int64 // how much bytes we read
}

func NewCachePub(f readerAtWriter) (c *CachePub) {
	c = &CachePub{f: f}
	c.cond = sync.NewCond(c.mu.RLocker())
	return
}

func (c *CachePub) Write(b []byte) (n int, e error) {
	n, e = c.f.Write(b)

	c.mu.Lock()

	c.n += int64(n)
	if e != nil {
		c.err = e
	}

	c.mu.Unlock()

	c.cond.Broadcast() // wake up readers

	return
}

func (c *CachePub) Finish() {
	c.mu.Lock()
	c.err = io.EOF
	c.mu.Unlock()

	c.cond.Broadcast() // wake them up

	// do not call f.Close(), as there may be readers still working on it.
	// rely on os.File finalizer to close handle
}

// should be invoked by writer, after finishing writing
// returns how much bytes it wrote to file
func (c *CachePub) Wrote() int64 {
	return c.n
}

func (c *CachePub) Error() (e error) {
	c.mu.RLock()
	e = c.err
	c.mu.RUnlock()
	return
}

func (c *CachePub) Cancel(err error) {
	if err == nil || err == io.EOF {
		panic("wrong cancel usage")
	}

	c.mu.Lock()
	c.err = err
	c.mu.Unlock()

	c.cond.Broadcast() // wake them up
}

func (c *CachePub) NewReader() *Reader {
	return &Reader{c: c}
}

// non-thread-safe
func (r *Reader) Read(b []byte) (n int, e error) {
	r.c.mu.RLock()

	for r.n >= r.c.n && r.c.err == nil {
		r.c.cond.Wait()
	}

	wn := r.c.n
	we := r.c.err

	r.c.mu.RUnlock()

	if r.n < wn {
		if int64(len(b)) > wn-r.n {
			b = b[:wn-r.n]
		}
		n, e = r.c.f.ReadAt(b, r.n)
		r.n += int64(n)
		if e != nil && e != io.EOF {
			return
		}
	}
	if r.n >= wn {
		e = we
	} else if e == io.EOF {
		e = io.ErrUnexpectedEOF
	}

	return
}

// non-thread-safe
// should be invoked by reader, after getting read error
// function name `Read` would conflict
// returns how much bytes it read from CachePub
func (r *Reader) FinishedReading() int64 {
	return r.n
}
