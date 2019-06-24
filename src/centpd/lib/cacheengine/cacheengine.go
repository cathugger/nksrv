package cacheengine

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"centpd/lib/cachepub"
	fu "centpd/lib/fileutil"
	"centpd/lib/xos"
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

type cacheObj struct {
	m         sync.RWMutex
	cond      *sync.Cond
	finished  bool
	finisherr error // file move error
	p         *cachepub.CachePub
}

type Backend interface {
	MakeFilename(objid string) string
	NewTempFile() (*os.File, error)
	Generate(w io.Writer, objid string, objinfo interface{}) error
}

type CacheEngine struct {
	m sync.RWMutex
	w map[string]*cacheObj
	b Backend
}

func NewCacheEngine(b Backend) CacheEngine {
	return CacheEngine{
		w: make(map[string]*cacheObj),
		b: b,
	}
}

type CopyDestination interface {
	CopyFrom(
		src io.Reader, objid string, objinfo interface{}) (
		written int64, err error)
}

func obtainFromCache(
	w CopyDestination, filename string, off int64,
	objid string, objinfo interface{}) (bool, error) {

	f, e := os.Open(filename)
	if e != nil {
		if os.IsNotExist(e) {
			e = nil
		}
		return false, e
	}
	defer f.Close()

	if off != 0 {
		_, e = f.Seek(off, 0)
		if e != nil {
			return true, e
		}
	}

	_, e = w.CopyFrom(f, objid, objinfo)

	return true, e
}

func (ce *CacheEngine) ObtainItem(
	w CopyDestination, objid string, objinfo interface{}) error {

	var fx *os.File
	var err error
	var o, oo *cacheObj
	var cpub *cachepub.CachePub
	var exists bool

	filename := ce.b.MakeFilename(objid)

	ce.m.RLock()
	o = ce.w[objid]
	ce.m.RUnlock()

	if o == nil {
		exists, err = obtainFromCache(w, filename, 0, objid, objinfo)
		if exists || err != nil {
			// if we finished successfuly, or failed in a way we cannot recover
			return err
		}
	} else {
		goto readExisting
	}

	// neither file nor wip object exist, so make new
	fx, err = ce.b.NewTempFile()
	if err != nil {
		return fmt.Errorf("failed making temporary file: %v", err)
	}
	cpub = cachepub.NewCachePub(fx)
	o = &cacheObj{p: cpub}
	o.cond = sync.NewCond(o.m.RLocker())

	ce.m.Lock()
	oo = ce.w[objid]
	// don't overwrite existing object
	// incase it was made while we weren't looking
	if oo == nil {
		ce.w[objid] = o
	}
	ce.m.Unlock()

	if oo != nil {
		// running generator exists
		// uhhh now we have to delete our existing thing..
		fn := fx.Name()
		fx.Close()
		os.Remove(fn)
		// and switch to what we got
		o = oo
		goto readExisting
	}

	// start generator
	// do et
	go func() {
		var we error // dangerous - don't confuse with outer err

		we = ce.b.Generate(cpub, objid, objinfo)
		if we != nil {
			if we == io.EOF {
				we = io.ErrUnexpectedEOF
			}
			cpub.Cancel(we)
		} else {
			cpub.Finish()
		}
		tn := fx.Name()
		// XXX maybe we should wait a little there so that readers could finish reading? idk
		// at this point file was written but if close fails,
		// readers (who didn't finish before close) won't be able to
		// read rest/reopen. therefore signal error
		e := fx.Close()
		if we == nil && e != nil {
			we = fmt.Errorf("worker failed closing file: %v", e)
		}
		if we == nil {
			// move from tmp to stable
			we = fu.RenameNoClobber(tn, filename)
			if os.IsExist(we) {
				we = nil
			}
			if we != nil {
				we = fmt.Errorf("worker failed renaming file: %v", we)
			}
		}
		if we != nil {
			os.Remove(tn)
		}
		// notify readers if any about availability
		o.m.Lock()
		o.finished = true
		o.finisherr = we
		o.m.Unlock()
		o.cond.Broadcast()
		// take out of map
		ce.m.Lock()
		delete(ce.w, objid)
		ce.m.Unlock()
	}()

readExisting:

	r := o.p.NewReader()
	done, err := w.CopyFrom(r, objid, objinfo)
	if !xos.IsClosed(err) {
		// nil(which would mean full success) or non-recoverable error
		if err == nil {
			// all done there no need to read file anymore
			return nil
		} else {
			return fmt.Errorf("w.CopyFrom err: %v", err)
		}
	}
	// file was closed
	// sanity check if it was actually written properly
	err = o.p.Error()
	if err != io.EOF {
		return fmt.Errorf(
			"CachePub in unexpected error state: %v", err)
	}
	// wait till file gets moved to stable storage
	// XXX maybe simpler design (spinlock, or finished being read in atomic way) would be better?
	o.m.RLock()
	for !o.finished {
		o.cond.Wait()
	}
	err = o.finisherr
	o.m.RUnlock()
	// check error from worker
	if err != nil {
		return fmt.Errorf("finisherr: %v", err)
	}
	// read from stable storage
	exists, err = obtainFromCache(w, filename, done, objid, objinfo)
	if exists || err != nil {
		// if we finished successfuly, or failed in a way we cannot recover
		if err == nil {
			return nil
		} else {
			return fmt.Errorf(
				"failed obtaining after generation: %v", err)
		}
	}
	// this shouldn't happen, but in theory could
	// ensure we print something meaningful in such weird race case
	return errors.New(
		"after generation obtainFromCache didn't find file")
}
