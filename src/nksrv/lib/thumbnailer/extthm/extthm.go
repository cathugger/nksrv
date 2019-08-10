package extthm

// thumbnailer which calls external programs and is designed to work for most of things with some amount of customization

import (
	"os"

	"github.com/gobwas/glob"

	"nksrv/lib/fstore"
	"nksrv/lib/thumbnailer"
)

/*
 * organization:
 * mime type => {
 *   scanner: {type, options},
 *   thumbnailer: {type, options},
 * }
 */

// this whole thumbnailer specific config
type Config struct {
	MaxWidth, MaxHeight int
	MaxPixels           int
	MaxFileSize         int64
}

// defaults for above
var DefaultConfig = Config{
	MaxWidth:    8192,
	MaxHeight:   8192,
	MaxPixels:   6144 * 6144,
	MaxFileSize: 64 * 1024 * 1024,
}

func (c Config) BuildThumbnailer(
	fs *fstore.FStore) (thumbnailer.Thumbnailer, error) {

	// XXX check existence of binaries

	return &ExternalThumbnailer{cfg: c, fs: fs}, nil
}

type tparams = map[string]string

type thmbackend interface {
	doThumbnailing(
		p tparams, f *os.File, ext, mimeType string, cfg thumbnailer.ThumbConfig) (
		res thumbnailer.ThumbResult, fi thumbnailer.FileInfo, err error)
}

type extExec struct {
	// matching either by mime or ext or both
	m_mime glob.Glob
	m_ext  glob.Glob
	// thing actually doing execution
	t thmbackend
	// params to pass to thumbnailer
	p tparams
}

type ExternalThumbnailer struct {
	cfg    Config
	fs     *fstore.FStore
	routes []extExec
}

func (t *ExternalThumbnailer) ThumbProcess(
	f *os.File, ext, mimeType string, cfg thumbnailer.ThumbConfig) (
	res thumbnailer.ThumbResult, fi thumbnailer.FileInfo, err error) {

	close_err := func() { err = f.Close() }

	if t.cfg.MaxFileSize > 0 {
		var st os.FileInfo
		st, err = f.Stat()
		if err != nil {
			_ = f.Close()
			return
		}
		if st.Size() > t.cfg.MaxFileSize {
			close_err()
			return
		}
	}

	for i := range t.routes {
		if (t.routes[i].m_mime == nil || t.routes[i].m_mime.Match(mimeType)) &&
			(t.routes[i].m_ext == nil || t.routes[i].m_ext.Match(ext)) {

			// XXX fallbacks?
			return t.routes[i].t.
				doThumbnailing(t.routes[i].p, f, ext, mimeType, cfg)
		}
	}

	// matched no routes - just fug off
	close_err()
	return
}
