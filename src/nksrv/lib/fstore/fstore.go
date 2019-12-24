package fstore

// abstracts and automates some filestore operations

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/xerrors"
)

type Config struct {
	Path    string
	Private string
}

type FStore struct {
	rootdir string // root folder + path separator if needed
	privdir string // priv (per instance) folder + path sep if needed
	privpfx string // priv pfx used for initDirs

	// pre-declared dirs
	declDirsMu sync.RWMutex
	declDirs   map[string]bool // whether already precreated
}

// tempfile logic based on stdlib' io/ioutil/tempfile.go

func badPrivate(p string) bool {
	return p == "" || (p[0] == '.' && p != ".") ||
		strings.ContainsAny(p, `/\ `)
}

func cleanWSlash(p string) string {
	p = path.Clean(p)
	if p != "." {
		if p == "/" {
			panic(`plz don't use "/"`)
		}
		return p + string(os.PathSeparator)
	} else {
		return ""
	}
}

const (
	newMainDirMode = 0777
	newDirMode     = 0700
	newFileMode    = 0666
)

func OpenFStore(cfg Config) (s FStore, err error) {

	s.declDirs = make(map[string]bool)

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

	if s.rootdir != "" {
		err = os.MkdirAll(s.rootdir[:len(s.rootdir)-1], newMainDirMode)
		if err != nil {
			return
		}
	}
	if s.privdir != "" {
		err = os.MkdirAll(s.privdir[:len(s.privdir)-1], newDirMode)
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

func (fs *FStore) declareDir(root, rpfx, dir string, preserve bool) (err error) {
	fs.declDirsMu.Lock()
	defer fs.declDirsMu.Unlock()

	// pre-mark
	fs.declDirs[rpfx+dir] = false

	if preserve {
		err = os.MkdirAll(root+dir, newDirMode)
		if err != nil {
			return
		}

		fs.declDirs[rpfx+dir] = true
	} else {
		err = os.RemoveAll(root + dir)
	}
	return
}

func (fs *FStore) DeclareDir(dir string, preserve bool) error {
	return fs.declareDir(fs.privdir, fs.privpfx, dir, preserve)
}

func (fs *FStore) DeclareGlobalDir(dir string, preserve bool) (err error) {
	return fs.declareDir(fs.rootdir, "", dir, preserve)
}

//var errNotDeclared = errors.New("directory not declared")

func (fs *FStore) ensureDir(fulldir, dir string) (err error) {
	// quick read-only peek first
	fs.declDirsMu.RLock()
	inited, exists := fs.declDirs[dir]
	fs.declDirsMu.RUnlock()

	// quick exit conditions determined from peek
	if inited {
		// should be already made
		return
	}
	if !exists {
		// do not allow stuff not in whitelist
		//return errNotDeclared
		panic("directory not declared: " + dir)
	}

	// we'll need to modify it...
	fs.declDirsMu.Lock()
	defer fs.declDirsMu.Unlock()

	// maybe it changed since we've looked...
	inited = fs.declDirs[dir]
	if inited {
		return
	}

	// perform mkdir
	err = os.MkdirAll(fulldir, newDirMode)
	if err != nil {
		return xerrors.Errorf("error at os.MkdirAll: %w", err)
	}

	// mark as inited
	fs.declDirs[dir] = true

	return
}

func (fs *FStore) makeRndFile(fulldir, pfx, ext string) (f *os.File, err error) {
	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(fulldir, pfx+nextSuffix()+ext)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, newFileMode)
		if os.IsExist(err) {
			nconflict++
			if nconflict > 10 {
				reseed()
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
		err = os.Mkdir(name, newDirMode)
		if os.IsExist(err) {
			nconflict++
			if nconflict > 10 {
				reseed()
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
