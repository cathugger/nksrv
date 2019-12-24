package fstore

// abstracts and automates some filestore operations

import (
	crand "crypto/rand"
	"encoding/binary"
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
	rootdir string // root folder + path separator if needed
	privdir string // priv (per instance) folder + path sep if needed
	privpfx string // priv pfx used for initDirs

	initDirsMu sync.Mutex
	initDirs   map[string]struct{}
}

// tempfile logic based on stdlib' io/ioutil/tempfile.go

var (
	rng     pcg.PCG64s
	rngInit bool
	rngMu   sync.Mutex
)

func reseed() {
	var b [16]byte
	if _, e := crand.Read(b[:]); e != nil {
		panic(e.Error())
	}
	x, y := binary.BigEndian.Uint64(b[:8]), binary.BigEndian.Uint64(b[8:])
	x += uint64(os.Getpid())
	y += uint64(time.Now().UnixNano())
	rng.Seed(x, y)
}

func nextSuffix() string {
	rngMu.Lock()
	if !rngInit {
		reseed()
		rngInit = true
	}
	x := rng.Bounded(1e18)
	rngMu.Unlock()
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
		s.privdir = rootdir + "_priv" + string(os.PathSeparator) + privdir
	} else {
		// was explicit "." which got converted to ""
		s.privdir = rootdir
	}
	s.rootdir = s.privdir[:len(rootdir)]
	s.privpfx = s.privdir[len(rootdir):]

	if s.privdir != "" {
		err = os.MkdirAll(s.privdir[:len(s.privdir)-1], 0777)
		if err != nil {
			return
		}
	}

	return
}

// Main returns main directory with slash if needed.
func (fs FStore) Main() string {
	return fs.rootdir
}

func (fs *FStore) makeDir(root, rpfx, dir string, mode os.FileMode) (err error) {
	fs.initDirsMu.Lock()
	defer fs.initDirsMu.Unlock()

	//fmt.Fprintf(os.Stderr, "newdir: %q\n", fs.root+dir)
	err = os.MkdirAll(root+dir, mode)
	if err != nil {
		return
	}
	// mark so if someone removes we'll err
	fs.initDirs[rpfx+dir] = struct{}{}
	return
}

func (fs *FStore) MakeDir(dir string, mode os.FileMode) error {
	return fs.makeDir(fs.privdir, fs.privpfx, dir, mode)
}

func (fs *FStore) MakeGlobalDir(dir string, mode os.FileMode) (err error) {
	return fs.makeDir(fs.rootdir, "", dir, mode)
}

func (fs *FStore) removeDir(root, rpfx, dir string) (err error) {
	fs.initDirsMu.Lock()
	defer fs.initDirsMu.Unlock()

	err = os.RemoveAll(root + dir)
	delete(fs.initDirs, rpfx+dir)
	return
}

func (fs *FStore) RemoveDir(dir string) error {
	return fs.removeDir(fs.privdir, fs.privpfx, dir)
}

func (fs *FStore) RemoveGlobalDir(dir string) error {
	return fs.removeDir(fs.rootdir, "", dir)
}

func (fs *FStore) ensureDir(fulldir, dir string) (err error) {
	fs.initDirsMu.Lock()
	defer fs.initDirsMu.Unlock()

	if _, inited := fs.initDirs[dir]; !inited {
		err = os.MkdirAll(fulldir, 0700)
		if err != nil {
			return fmt.Errorf("error at os.MkdirAll: %v", err)
		}
		fs.initDirs[dir] = struct{}{}
	}

	return
}

func (fs *FStore) makeRndFile(fulldir, pfx, ext string) (f *os.File, err error) {
	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(fulldir, pfx+nextSuffix()+ext)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if os.IsExist(err) {
			nconflict++
			if nconflict > 10 {
				rngMu.Lock()
				reseed()
				rngMu.Unlock()
			}
			continue
		}
		break
	}
	return
}

func (fs *FStore) makeRndDir(fulldir, pfx, ext string) (name string, err error) {
	nconflict := 0
	for i := 0; i < 10000; i++ {
		name = filepath.Join(fulldir, pfx+nextSuffix()+ext)
		err = os.Mkdir(name, 0700)
		if os.IsExist(err) {
			nconflict++
			if nconflict > 10 {
				rngMu.Lock()
				reseed()
				rngMu.Unlock()
			}
			continue
		}
		break
	}
	return
}

func (fs *FStore) newFile(
	root, rpfx, dir, pfx, ext string) (f *os.File, err error) {

	fulldir := root + dir

	err = fs.ensureDir(fulldir, rpfx+dir)
	if err != nil {
		return
	}

	return fs.makeRndFile(fulldir, pfx, ext)
}

func (fs *FStore) NewFile(dir, pfx, ext string) (*os.File, error) {
	return fs.newFile(fs.privdir, fs.privpfx, dir, pfx, ext)
}

func (fs *FStore) NewGlobalFile(dir, pfx, ext string) (*os.File, error) {
	return fs.newFile(fs.rootdir, "", dir, pfx, ext)
}

const tmpDir = "_tmp"

func (fs *FStore) TempFile(pfx, ext string) (f *os.File, err error) {
	return fs.NewFile(tmpDir, pfx, ext)
}

func (fs *FStore) newDir(
	root, rpfx, dir, pfx, ext string) (name string, err error) {

	fulldir := root + dir

	err = fs.ensureDir(fulldir, rpfx+dir)
	if err != nil {
		return
	}

	return fs.makeRndDir(fulldir, pfx, ext)
}

func (fs *FStore) NewDir(dir, pfx, ext string) (string, error) {
	return fs.newDir(fs.privdir, fs.privpfx, dir, pfx, ext)
}

func (fs *FStore) NewGlobalDir(dir, pfx, ext string) (string, error) {
	return fs.newDir(fs.rootdir, "", dir, pfx, ext)
}
