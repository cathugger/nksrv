package cacheengine

import (
	"errors"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/xerrors"

	"nksrv/lib/utils/cachepub"
	fu "nksrv/lib/utils/fs/fileutil"
	"nksrv/lib/utils/fs/xos"
)

// this gon be fuken weird and complicated im telling u

// obtain is hybrid get-cached-or-make-on-the-spot operation
// to minimise latency, we should be able to begin submission of document to requester(s) before generation operation isn't yet finished
// to achieve that in efficient and relatively simple way, generation is output to single file handle opened in read-write mode
// then pread is used by reader who initiated operation, and by any reader(s) who also requested result of operation before it was finished
// when generation is done, file handle is forcefuly closed (so that no slow reader(s) could block the following)
//   and file is moved to stable storage
// if forceful close interrupted any ongoing pread operation of reader(s),
//   they then shall independently open file moved to stable storage, seek and proceed reading remaining data
// this design should be safe to use in operating systems, where opening in write mode imply obtaining exclusive lock of file (Windows)

// when doing map locking, assume that generation operation takes longer than filesystem check of existing cached object,
//   and do not hold lock during check
// this should improve concurrency, and hopefuly not cause much harm in case assumption isn't true and object is regenerated

// access pattern:
// every user wanting to read any current value must rlock
// every user wanting to write to finished/finisherr/p/f/n, must wlock
// if n == 0, object must be taken off active object pool
// n==0 on enter CAN happen if object is being removed from pool; in this case, wait on cond and try taking off new object after
type CacheObj struct {
	m sync.RWMutex // guard for all
	c sync.Cond    // cond for reading new stuff

	n int                // current amount of readers (refcount for gc)
	f *os.File           // for shared reading
	p *cachepub.CachePub // for shared reading while writing
	e error              // error
}

type sharedReader struct {
	f *os.File
	n int64
}

func (r *sharedReader) Read(b []byte) (n int, err error) {
	n, err = r.f.ReadAt(b, r.n)
	if n < 0 {
		panic("negative read")
	}
	r.n += int64(n)
	return
}

type Backend interface {
	MakeFilename(objid string) string
	NewTempFile() (*os.File, error)
	Generate(w io.Writer, objid string, objinfo interface{}) error
}

type CacheEngine struct {
	m sync.RWMutex
	w map[string]*CacheObj
	b Backend
}

func NewCacheEngine(b Backend) CacheEngine {
	return CacheEngine{
		w: make(map[string]*CacheObj),
		b: b,
	}
}

type CopyDestination interface {
	CopyFrom(
		src io.Reader, objid string, objinfo interface{}) (
		written int64, err error)
}

var errRestart = errors.New("plz restart kthxbai")

func (ce *CacheEngine) ObtainItem(
	w CopyDestination, objid string, objinfo interface{}) error {

	var o, oo *CacheObj

	var nx int
	var fx *os.File
	var px *cachepub.CachePub
	var ex error

	var done int64
	var r io.Reader
	var wg sync.WaitGroup
	var cp_read_done int32

	filename := ce.b.MakeFilename(objid)

	// expects ce.w to be locked in some way, and o to point to current object
	register_to_o := func() {
		o.m.Lock()
		if o.n > 0 {
			o.n++
		}
		nx, px, fx, ex = o.n, o.p, o.f, o.e
		o.m.Unlock()
	}

	// expects ce.w to be unlocked, and o to point to current object
	unregister_from_o := func() {
		o.m.Lock()
		if o.n > 0 {
			o.n--
			nx, fx = o.n, o.f
		} else {
			nx = -1
		}
		o.m.Unlock()
		// GC
		if nx == 0 {
			// if nx == 0 it's all ours as no other thread can have this value
			ce.m.Lock()
			if o == ce.w[objid] {
				// take out
				delete(ce.w, objid)
			}
			ce.m.Unlock()

			if fx != nil {
				fx.Close()

				o.m.Lock()
				o.f = nil
				o.m.Unlock()
			}
		}
	}

	// expects ce.w to be unlocked, marks oo as broken and takes it out
	oo_broken := func(el error) {
		// take it out
		ce.m.Lock()
		if oo == ce.w[objid] {
			delete(ce.w, objid)
		}
		ce.m.Unlock()

		// mark as broke
		oo.m.Lock()
		oo.e = el // propagate error
		oo.n = 0  // not suitable for usage anymore
		oo.m.Unlock()

		// signal
		oo.c.Broadcast()
	}

begin:
	// first time check
	ce.m.RLock()
	o = ce.w[objid]
	if o != nil {
		register_to_o()
	}
	ce.m.RUnlock()

	if o != nil && nx <= 0 {
		// if it's got 0 users we aint going to use it as it's being closed
		runtime.Gosched()
		goto begin
	}

	if o == nil {
		// new cache obj
		oo = &CacheObj{n: 1}
		oo.c.L = oo.m.RLocker()

		// second time check & set
		ce.m.Lock()
		o = ce.w[objid]
		if o == nil {
			ce.w[objid] = oo
		} else {
			register_to_o()
		}
		ce.m.Unlock()

		if o == nil {
			// entry didn't exist previously, we successfuly set it

			o = oo

			// first attempt to open existing passive file
			fx, ex = os.Open(filename)
			if ex != nil {
				if !os.IsNotExist(ex) {
					ex = xerrors.Errorf("failed opening passive file: %w", ex)
					oo_broken(ex)
					return ex
				}

				// passive file doesn't exist, so we'll generate new one
				fx, ex = ce.b.NewTempFile()
				if ex != nil {
					ex = xerrors.Errorf("failed making temporary file: %w", ex)
					oo_broken(ex)
					return ex
				}

				// put up cachepub
				px = cachepub.NewCachePub(fx)
				oo.m.Lock()
				oo.p = px
				oo.m.Unlock()
				oo.c.Broadcast() // let readers know that we now have cachepub

				// start generator
				// do et
				wg.Add(1)
				go func() {
					var we error // dangerous - don't confuse with outer err

					we = ce.b.Generate(px, objid, objinfo)
					if we != nil {
						we = xerrors.Errorf("worker failed generating: %w", we)
						px.Cancel(we)
					} else {
						px.Finish()
					}
					tn := fx.Name()
					// XXX maybe we should wait a little there so that readers could finish reading? idk
					// at this point file was written but if close fails,
					// readers (who didn't finish before close) won't be able to
					// read rest/reopen. therefore signal error
					// XXX new readers AFTER this will probably get "file already closed"
					// hopefuly that's guaranteed
					cle := fx.Close()
					// once closed, it'll become useless. don't let new readers stumble upon it
					oo.m.Lock()
					oo.p = nil
					oo.m.Unlock()
					// don't notify yet, do that once we've opened file for reading
					if we == nil && cle != nil {
						we = xerrors.Errorf("worker failed closing file: %w", cle)
					}
					if we == nil {
						// move from tmp to stable
						we = fu.RenameNoClobber(tn, filename)
						if os.IsExist(we) {
							we = nil
						}
						if we != nil {
							we = xerrors.Errorf("worker failed renaming file: %w", we)
						}
					}
					if we != nil {
						os.Remove(tn)
					}
					if we == nil {
						if atomic.LoadInt32(&cp_read_done) != 0 {
							dontRead := false
							// if reader already finished
							oo.m.Lock()
							// and it was only one reader
							if oo.n == 1 {
								// set err to restart because new threads may pick
								oo.e = errRestart
								dontRead = true
							}
							oo.m.Unlock()
							// skip opening file for reading
							if dontRead {
								oo.c.Broadcast()
								wg.Done()
								return
							}
						}
						// open file for reading
						fx, we = os.Open(filename)
						if we != nil {
							// not supposed to happen - file we've just closed is gone
							we = xerrors.Errorf("worker failed reopening file: %w", we)
						}
					}
					if we != nil {
						// error
						fx = nil
						oo_broken(we)
					} else {
						// done ok
						oo.m.Lock()
						oo.f = fx
						oo.m.Unlock()
						// notify readers if any about availability
						oo.c.Broadcast()
					}
					wg.Done()
				}()

				goto feedFromCachePub
			} else {
				// err == nil, we've successfuly opened file for reading

				oo.m.Lock()
				oo.f = fx
				oo.m.Unlock()

				oo.c.Broadcast()

				goto feedFromReadFile
			}
			// never reached
		}
	}

	// o != nil, either first time or second time

	// entry exists wait (if needed) until it has something to say
	if px == nil && fx == nil && ex == nil {
		o.m.RLock()
		for o.p == nil && o.f == nil && o.e == nil {
			o.c.Wait()
		}
		px, fx, ex = o.p, o.f, o.e
		o.m.RUnlock()
	}

	if ex != nil {
		// it signals error ugh
		unregister_from_o()
		if ex == errRestart {
			goto begin
		}
		return xerrors.Errorf("reported object error: %w", ex)
	}
	if fx != nil {
		goto feedFromReadFile
	}
	// px != nil

feedFromCachePub:
	r = px.NewReader()

	done, ex = w.CopyFrom(r, objid, objinfo)

	if !xos.IsClosed(ex) {

		atomic.StoreInt32(&cp_read_done, 1)

		// nil(which would mean full success) or non-recoverable error
		if ex != nil {
			ex = xerrors.Errorf("cachepub consumption error: %w", ex)
		}

		wg.Wait() // ensure writer thread is done

		unregister_from_o()

		return ex
	}
	// file was closed
	// sanity check if it was actually written properly
	ex = px.Error()
	if ex != io.EOF {
		wg.Wait() // ensure writer thread is done

		unregister_from_o()

		return xerrors.Errorf(
			"CachePub in unexpected error state: %w", ex)
	}

	o.m.RLock()
	for o.f == nil && o.e == nil {
		o.c.Wait()
	}
	fx, ex = o.f, o.e
	o.m.RUnlock()

	if ex != nil {
		unregister_from_o()
		return xerrors.Errorf("reported object error: %w", ex)
	}

feedFromReadFile:
	r = &sharedReader{f: fx, n: done}
	_, ex = w.CopyFrom(r, objid, objinfo)
	if ex != nil {
		ex = xerrors.Errorf("sharedreader consumption error: %w", ex)
	}
	unregister_from_o()
	return ex
}

/* notes about hypothetical
 * "deleted but not commited" "commited but interrupted before delete"
 * case:
 * we should probably delete and park placeholder to deny making new cache entries
 * after stuff gets commited, signal "gone" status
 * this way we won't leave files even after unclean shutdown after commit but before delete
 */

func (ce *CacheEngine) RemoveItemStart(objid string) (n *CacheObj, err error) {
	// XXX how do we handle remove denial because of active readers?
	// could inject fake obj into w and spin until last reader finishes
	// but it won't come up on any system other than windows probably

	filename := ce.b.MakeFilename(objid)

	n = &CacheObj{n: 1}
	n.c.L = n.m.RLocker()

	ce.m.Lock()
	o := ce.w[objid]
	ce.w[objid] = n // overwrite existing if any
	ce.m.Unlock()

	// XXX fails in multiprocess model
	if !safeToRemoveOpen && o != nil {
		// we have already existing, wait till it gets cleared

		// eventually timeout tho
		go deleteTimeoutClose(o)

		o.m.RLock()
		for (o.n > 0 || o.f != nil) && o.e == nil {
			o.c.Wait()
		}
		o.m.RUnlock()
		// okay it should be finished at this point
	}

	err = os.Remove(filename)
	if err != nil && os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		// we've failed so don't hold lock as file is still on disk
		ce.RemoveItemFinish(objid, n)
	}
	return
}

func deleteTimeoutClose(o *CacheObj) {

	time.Sleep(5 * time.Second)

	o.m.Lock()

	if o.n <= 0 {
		// already closed or inactive
		o.m.Unlock()
		return
	}

	if o.f != nil {
		// close, mark as closed, signal error
		o.f.Close()
		o.f = nil
		o.n = 0
		o.e = errors.New("forcibly closed because of prune timeout")

		o.m.Unlock()
		o.c.Broadcast()
		return
	}

	// at this point 2 possibilities:
	// it's placeholder entry
	// or it's generating and not reading yet
	// either way extend timeout because we dont know what to do
	// and both conditions are supposed to end eventually
	o.m.Unlock()
	go deleteTimeoutClose(o) // rethrow self
}

func (ce *CacheEngine) RemoveItemFinish(objid string, oo *CacheObj) {
	ce.m.Lock()
	o := ce.w[objid]
	if o == oo {
		// only DELET if its ours
		delete(ce.w, objid)
	}
	ce.m.Unlock()

	if o == nil {
		return
	}

	if o.p != nil || o.f != nil || o.e != nil {
		// we can't be overwritten by valid object
		// if we're overwritten, that has to be placeholder object too
		panic("wrong object condition")
	}

	// NOTE: we're doing operations with original object not the one we got
	// mark as done & ded
	oo.m.Lock()
	oo.e = errRestart
	oo.n = 0
	oo.m.Unlock()
	// notify readers if any about (un)availability
	oo.c.Broadcast()

	// XXX
	// if running multiprocess locking might have not helped;
	// further deletion success is not guaranteed either because
	// we could be interrupted before this point, so delet there
	// would be oppurtunistic.
	// is likelihood of this happening worth it?
	// if something cleans the cache for us it'd pick it up anyway..
}

func (ce *CacheEngine) RemoveItem(objid string) (err error) {
	oo, err := ce.RemoveItemStart(objid)
	if err == nil {
		ce.RemoveItemFinish(objid, oo)
	}
	return
}
