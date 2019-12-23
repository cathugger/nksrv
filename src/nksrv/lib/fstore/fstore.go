package fstore

// abstracts and automates some filestore operations

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"nksrv/lib/pcg"
)

type Config struct {
	Path string
}

type FStore struct {
	root     string // root folder + path separator
	initMu   sync.Mutex
	initDirs map[string]struct{}
}

// tempfile logic based on stdlib' io/ioutil/tempfile.go

var (
	rand     pcg.PCG64s
	randInit bool
	randMu   sync.Mutex
)

func reseed() {
	rand.Seed(uint64(os.Getpid()), uint64(time.Now().UnixNano()))
}

func nextSuffix() string {
	randMu.Lock()
	if !randInit {
		reseed()
		randInit = true
	}
	x := rand.Bounded(1e18)
	randMu.Unlock()
	return strconv.FormatUint(1e18+x, 10)[1:]
}

const tmpDir = "_tmp"

func OpenFStore(cfg Config) (s FStore, e error) {
	s.initDirs = make(map[string]struct{})

	i := len(cfg.Path)
	if i > 0 && !os.IsPathSeparator(cfg.Path[i-1]) {
		s.root = cfg.Path + string(os.PathSeparator)
	} else {
		s.root = cfg.Path
	}
	if i > 0 {
		e = os.MkdirAll(s.root[:len(s.root)-1], 0777)
		if e != nil {
			return
		}
	}

	return
}

func (fs FStore) Main() string {
	return fs.root
}

func (fs *FStore) MakeDir(dir string) (err error) {
	fs.initMu.Lock()
	defer fs.initMu.Unlock()

	//fmt.Fprintf(os.Stderr, "newdir: %q\n", fs.root+dir)

	err = os.MkdirAll(fs.root+dir, 0700)
	if err != nil {
		return
	}
	fs.initDirs[dir] = struct{}{}

	return
}

func (fs *FStore) RemoveDir(dir string) (e error) {
	fs.initMu.Lock()
	e = os.RemoveAll(fs.root + dir)
	delete(fs.initDirs, dir)
	fs.initMu.Unlock()
	return
}

func (fs *FStore) CleanTemp() error {
	return fs.RemoveDir(tmpDir)
}

func (fs *FStore) ensureDir(fulldir, dir string) (err error) {
	fs.initMu.Lock()
	defer fs.initMu.Unlock()

	if _, inited := fs.initDirs[dir]; !inited {
		err = os.MkdirAll(fulldir, 0700)
		if err != nil {
			return fmt.Errorf("error at os.MkdirAll: %v", err)
		}
		fs.initDirs[dir] = struct{}{}
	}

	return
}

func (fs *FStore) NewFile(dir, pfx, ext string) (f *os.File, err error) {
	fulldir := fs.root + dir

	err = fs.ensureDir(fulldir, dir)
	if err != nil {
		return
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(fulldir, pfx+nextSuffix()+ext)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if os.IsExist(err) {
			nconflict++
			if nconflict > 10 {
				randMu.Lock()
				reseed()
				randMu.Unlock()
			}
			continue
		}
		break
	}
	return
}

func (fs *FStore) TempFile(pfx, ext string) (f *os.File, err error) {
	return fs.NewFile(tmpDir, pfx, ext)
}
