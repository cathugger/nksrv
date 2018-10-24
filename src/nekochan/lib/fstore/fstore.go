package fstore

// abstracts and automates some filestore operations

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Config struct {
	Path string
}

type FStore struct {
	root     string
	initMu   sync.Mutex
	initDirs map[string]struct{}
}

// tempfile logic based on stdlib' io/ioutil/tempfile.go

var (
	rand   uint32
	randmu sync.Mutex
)

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextSuffix() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

const tmpDir = "_tmp"

func OpenFStore(cfg Config) (s FStore, _ error) {
	s.initDirs = make(map[string]struct{})

	i := len(cfg.Path)
	if i > 0 && !os.IsPathSeparator(cfg.Path[i-1]) {
		s.root = cfg.Path + string(os.PathSeparator)
	} else {
		s.root = cfg.Path
	}
	if i > 0 {
		e := os.MkdirAll(s.root[:i-1], 0777)
		if e != nil {
			return FStore{}, e
		}
	}

	return s, nil
}

func (fs FStore) Main() string {
	return fs.root
}

func (fs *FStore) CleanDir(dir string) (e error) {
	fs.initMu.Lock()
	e = os.RemoveAll(fs.root + dir)
	delete(fs.initDirs, dir)
	fs.initMu.Unlock()
	return
}

func (fs *FStore) CleanTemp() error {
	return fs.CleanDir(tmpDir)
}

func (fs *FStore) NewFile(dir, pfx, ext string) (f *os.File, err error) {
	fulldir := fs.root + dir

	fs.initMu.Lock()
	if _, inited := fs.initDirs[dir]; !inited {
		err = os.MkdirAll(fulldir, 0700)
		if err != nil {
			fs.initMu.Unlock()
			return nil, fmt.Errorf("error at os.MkdirAll: %v", err)
		}
		fs.initDirs[dir] = struct{}{}
	}
	fs.initMu.Unlock()

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(fulldir, pfx+nextSuffix()+ext)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
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
