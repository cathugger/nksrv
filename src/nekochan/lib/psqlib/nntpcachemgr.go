package psqlib

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/crypto/blake2s"

	"nekochan/lib/cachepub"
	fu "nekochan/lib/fileutil"
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

type nntpCacheObj struct {
	m        sync.RWMutex
	cond     *sync.Cond
	finished bool
	p        *cachepub.CachePub
}

type nntpCacheMgr struct {
	m        sync.RWMutex
	wipCache map[CoreMsgIDStr]*nntpCacheObj
}

func newNNTPCacheMgr() nntpCacheMgr {
	return nntpCacheMgr{
		wipCache: make(map[CoreMsgIDStr]*nntpCacheObj),
	}
}

func obtainFromCache(w nntpCopyer, filename string, off int64, num uint64, msgid CoreMsgIDStr) (bool, error) {
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

	_, e = w.Copy(num, msgid, f)

	return true, e
}

func (sp *PSQLIB) makeFilename(id CoreMsgIDStr) string {
	// id can contain invalid chars like /
	// we could just base32 id itself but that would allow it to grow over common file name limit of 255
	// so do blake2s
	idsum := blake2s.Sum256(unsafeStrToBytes(string(id)))
	enc := lowerBase32HexEnc.EncodeToString(idsum[:])
	return sp.nntpfs.Main() + enc + ".eml"
}

func (sp *PSQLIB) nntpObtainItemByMsgID(w nntpCopyer, cs *ConnState, msgid CoreMsgIDStr) error {
	var bid boardID
	var pid postID

	err := sp.db.DB.
		QueryRow("SELECT bid,pid FROM ib0.posts WHERE msgid = $1 LIMIT 1", string(msgid)).
		Scan(&bid, &pid)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.sqlError("posts row query scan", err)
	}
	num := artnumInGroup(cs, bid, pid)
	return sp.nntpObtainItemOrStat(w, num, msgid)
}

func (sp *PSQLIB) nntpObtainItemByNum(w nntpCopyer, cs *ConnState, num uint64) error {
	gs := getGroupState(cs)
	if gs == nil {
		return errNoBoardSelected
	}

	var msgid CoreMsgIDStr
	err := sp.db.DB.
		QueryRow("SELECT msgid FROM ib0.posts WHERE bid = $1 AND pid = $2 LIMIT 1", gs.bid, num).
		Scan(&msgid)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.sqlError("posts row query scan", err)
	}
	return sp.nntpObtainItemOrStat(w, num, msgid)
}

func (sp *PSQLIB) nntpObtainItemByCurr(w nntpCopyer, cs *ConnState) error {
	gs := getGroupState(cs)
	if gs == nil {
		return errNoBoardSelected
	}
	if gs.pid <= 0 {
		return errNotExist
	}

	var msgid CoreMsgIDStr
	err := sp.db.DB.
		QueryRow("SELECT msgid FROM ib0.posts WHERE bid = $1 AND pid = $2 LIMIT 1", gs.bid, gs.pid).
		Scan(&msgid)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.sqlError("posts row query scan", err)
	}
	return sp.nntpObtainItemOrStat(w, gs.pid, msgid)
}

func (sp *PSQLIB) nntpObtainItemOrStat(w nntpCopyer, num uint64, msgid CoreMsgIDStr) error {
	if _, ok := w.(statNNTPCopyer); !ok {
		return sp.nntpObtainItem(w, num, msgid)
	} else {
		// interface abuse
		_, err := w.Copy(num, msgid, nil)
		return err
	}
}

func (sp *PSQLIB) nntpObtainItem(w nntpCopyer, num uint64, msgid CoreMsgIDStr) error {
	var f *os.File
	var err error
	var o, oo *nntpCacheObj
	var cpub *cachepub.CachePub

	c := &sp.nntpmgr

	filename := sp.makeFilename(msgid)

	c.m.RLock()
	o = c.wipCache[msgid]
	c.m.RUnlock()

	if o == nil {
		exists, err := obtainFromCache(w, filename, 0, num, msgid)
		if exists || err != nil {
			// if we finished successfuly, or failed in a way we cannot recover
			return err
		}
	} else {
		goto readExisting
	}

	// neither file nor wip object exist, so make new
	f, err = sp.nntpfs.TempFile("", "")
	if err != nil {
		return fmt.Errorf("failed making temporary file: %v")
	}
	cpub = cachepub.NewCachePub(f)
	o = &nntpCacheObj{p: cpub}
	o.cond = sync.NewCond(o.m.RLocker())

	c.m.Lock()
	oo = c.wipCache[msgid]
	if oo == nil {
		c.wipCache[msgid] = o
	}
	c.m.Unlock()

	if oo != nil {
		// uhhh now we have to delete our existing thing..
		fn := f.Name()
		f.Close()
		os.Remove(fn)
		// and switch to what we got
		o = oo
		goto readExisting
	}

	// do et
	go func() {
		err = sp.nntpGenerate(cpub, num, msgid)
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			cpub.Cancel(err)
		} else {
			cpub.Finish()
		}
		tn := f.Name()
		// XXX maybe we should wait a little there so that readers could finish reading? idk
		f.Close()
		if err == nil {
			// move from tmp to stable
			fu.RenameNoClobber(tn, filename)
		} else {
			os.Remove(tn)
		}
		// notify readers if any about availability
		o.m.Lock()
		o.finished = true
		o.m.Unlock()
		o.cond.Broadcast()
		// take out of map
		c.m.Lock()
		delete(c.wipCache, msgid)
		c.m.Unlock()
	}()

readExisting:

	r := o.p.NewReader()
	done, err := w.Copy(num, msgid, r)
	if err != os.ErrClosed {
		// nil(which would mean full success) or non-recoverable error
		return err
	}
	// file was closed
	// sanity check if it was actually written properly
	err = o.p.Error()
	if err != io.EOF {
		return fmt.Errorf("CachePub in unexpected error state: %v", err)
	}
	// wait till file gets moved to stable storage
	// XXX maybe simpler design (spinlock, or finished being read in atomic way) would be better?
	o.m.RLock()
	for !o.finished {
		o.cond.Wait()
	}
	o.m.RUnlock()
	// read from stable storage
	exists, err := obtainFromCache(w, filename, done, num, msgid)
	if exists || err != nil {
		// if we finished successfuly, or failed in a way we cannot recover
		return err
	}
	// this shouldn't happen
	return errors.New("couldn't open file in stable storage even though it should be there")
}
