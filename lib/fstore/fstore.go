package fstore

// abstracts and automates some filestore operations

import (
	"../fstorecfg"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type FStore struct {
	root     string
	initTemp bool
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

func OpenFStore(cfg fstorecfg.ConfigFStore) (FStore, error) {
	var s FStore
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

	// cleanup tmpdir
	os.RemoveAll(s.root + "_tmp")

	return s, nil
}

func (fs *FStore) TempFile(pfx, ext string) (f *os.File, err error) {
	if !fs.initTemp {
		err = os.MkdirAll(fs.root+"_tmp", 0700)
		if err != nil {
			return nil, fmt.Errorf("error at os.MkdirAll: %v", err)
		}
		// not mp-safe but multiple mkdir's wont hurt us
		fs.initTemp = true
	}
	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(fs.root+"_tmp", pfx+nextSuffix()+ext)
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
