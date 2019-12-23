package fstore

// abstracts and automates some filestore operations

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"nksrv/lib/pcg"
)

type Config struct {
	Path    string
	Private string
}

type FStore struct {
	root     string // root folder + path separator
	private  string // private this instance folder
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

func badPrivate(p string) bool {
	return p == "" || (p[0] == '.' && p != ".") ||
		strings.ContainsAny(p, "/\\")
}

func cleanWSlash(p string) string {
	p = path.Clean(p)
	if p != "." {
		return p + string(os.PathSeparator)
	} else {
		return ""
	}
}

func OpenFStore(cfg Config) (s FStore, err error) {
	s.initDirs = make(map[string]struct{})

	if cfg.Path == "" || badPrivate(cfg.Private) {
		panic("incomplete/invalid config")
	}

	rootdir := cleanWSlash(cfg.Path)
	privdir := cleanWSlash(cfg.Private)
	if privdir != "" {
		s.private = rootdir + "_priv" + string(os.PathSeparator) + privdir
	} else {
		// was explicit "." which got converted to ""
		s.private = rootdir
	}
	s.root = s.private[:len(rootdir)]

	err = os.MkdirAll(s.private[:len(s.private)-1], 0777)
	if err != nil {
		return
	}

	return
}

// Main returns main directory with slash if needed.
func (fs FStore) Main() string {
	return fs.root
}

func (fs *FStore) MakeDir(dir string, mode os.FileMode) (err error) {
	fs.initMu.Lock()
	defer fs.initMu.Unlock()

	//fmt.Fprintf(os.Stderr, "newdir: %q\n", fs.root+dir)
	err = os.MkdirAll(fs.private+dir, mode)
	if err != nil {
		return
	}
	// mark so if someone removes we'll err
	fs.initDirs[fs.private[len(fs.root):]+dir] = struct{}{}
	return
}

func (fs *FStore) MakeGlobalDir(dir string, mode os.FileMode) (err error) {
	fs.initMu.Lock()
	defer fs.initMu.Unlock()

	err = os.MkdirAll(fs.root+dir, mode)
	if err != nil {
		return
	}
	// mark so if someone removes we'll err
	fs.initDirs[dir] = struct{}{}
	return
}

func (fs *FStore) RemoveDir(dir string) (err error) {
	fs.initMu.Lock()
	defer fs.initMu.Unlock()

	err = os.RemoveAll(fs.private + dir)
	delete(fs.initDirs, fs.private[len(fs.root):]+dir)
	return
}

func (fs *FStore) RemoveGlobalDir(dir string) (err error) {
	fs.initMu.Lock()
	defer fs.initMu.Unlock()

	err = os.RemoveAll(fs.root + dir)
	delete(fs.initDirs, dir)
	return
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

func (fs *FStore) newFile(root, dir, pfx, ext string) (f *os.File, err error) {
	fulldir := root + dir

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

func (fs *FStore) NewFile(dir, pfx, ext string) (*os.File, error) {
	return fs.newFile(fs.private, dir, pfx, ext)
}

func (fs *FStore) NewGlobalFile(dir, pfx, ext string) (*os.File, error) {
	return fs.newFile(fs.root, dir, pfx, ext)
}

const tmpDir = "_tmp"

func (fs *FStore) TempFile(pfx, ext string) (f *os.File, err error) {
	return fs.NewFile(tmpDir, pfx, ext)
}

func (fs *FStore) newDir(root, dir, pfx, ext string) (name string, err error) {
	fulldir := root + dir

	err = fs.ensureDir(fulldir, dir)
	if err != nil {
		return
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name = filepath.Join(fulldir, pfx+nextSuffix()+ext)
		err = os.Mkdir(name, 0700)
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

func (fs *FStore) NewDir(dir, pfx, ext string) (string, error) {
	return fs.newDir(fs.private, dir, pfx, ext)
}

func (fs *FStore) NewGlobalDir(dir, pfx, ext string) (string, error) {
	return fs.newDir(fs.root, dir, pfx, ext)
}
