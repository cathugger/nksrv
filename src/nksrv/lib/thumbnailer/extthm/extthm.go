package extthm

// thumbnailer which calls external programs and is designed to work for most of things with some amount of customization

import (
	"mime"
	"os"

	"github.com/gobwas/glob"

	"nksrv/lib/thumbnailer"
	"nksrv/lib/utils/emime"
	"nksrv/lib/utils/fs/fstore"
	. "nksrv/lib/utils/logx"
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
	fs *fstore.FStore, lx LoggerX) (_ thumbnailer.Thumbnailer, err error) {

	// XXX customization
	imt := new(magickBackend)
	// XXX "gm" for graphicsmagick
	err = imt.init("magick")
	if err != nil {
		return
	}

	mt := new(ffmpegSoxBackend)
	err = mt.init("ffprobe", "ffmpeg", "sox")
	if err != nil {
		return
	}

	t := &ExternalThumbnailer{
		cfg: c,
		fs:  fs,
		log: NewLogToX(lx, "extthm"),
		routes: []extExec{
			{t: imt, m_mime: glob.MustCompile("image/jpeg")},
			{t: imt, m_mime: glob.MustCompile("image/png")},
			{t: imt, m_mime: glob.MustCompile("image/gif")},
			{t: imt, m_mime: glob.MustCompile("image/webp")},
			{t: imt, m_mime: glob.MustCompile("image/bmp")},
			//{t: imt, m_mime: glob.MustCompile("image/avif")},
			//{t: imt, m_mime: glob.MustCompile("image/jxl")},

			{t: mt, m_mime: glob.MustCompile("video/webm"), p: tparams{"fmt": "webm"}},
			{t: mt, m_mime: glob.MustCompile("video/ogg"), p: tparams{"fmt": "ogg"}},
			{t: mt, m_mime: glob.MustCompile("video/mp4"), p: tparams{"fmt": "mp4"}},
			{t: mt, m_mime: glob.MustCompile("audio/ogg"), p: tparams{"fmt": "ogg"}},
			{t: mt, m_mime: glob.MustCompile("audio/mpeg"), p: tparams{"fmt": "mp3"}},
			{t: mt, m_mime: glob.MustCompile("audio/mp4"), p: tparams{"fmt": "mp4"}},
			{t: mt, m_mime: glob.MustCompile("audio/wave"), p: tparams{"fmt": "wav"}},
			{t: mt, m_mime: glob.MustCompile("audio/wav"), p: tparams{"fmt": "wav"}},
			{t: mt, m_mime: glob.MustCompile("audio/flac"), p: tparams{"fmt": "flac"}},
			{t: mt, m_mime: glob.MustCompile("audio/webm"), p: tparams{"fmt": "webm"}},
		},
	}

	imt.t = t
	mt.t = t

	return t, nil
}

type tparams = map[string]string

type thmbackend interface {
	doThumbnailing(
		p tparams, f *os.File, ext, mimeType string, fsize int64,
		cfg thumbnailer.ThumbConfig) (
		res thumbnailer.ThumbResult, err error)
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
	log    Logger
	routes []extExec
}

func (t *ExternalThumbnailer) ThumbProcess(
	f *os.File, ext, mimeType string, fsize int64,
	cfg thumbnailer.ThumbConfig) (
	res thumbnailer.ThumbResult, err error) {

	close_err := func() { err = f.Close() }

	mt, _, _ := mime.ParseMediaType(mimeType)

	t.log.LogPrintf(DEBUG, "ThumbProcess ext %q mime %q", ext, mt)

	if mt == "application/octet-stream" {
		nmt := emime.MIMECanonicalTypeByExtension(ext)
		if nmt == "" {
			nmt = emime.MIMETypeByExtension(ext)
		}
		if nmt != "" {
			mt = nmt
		}
	}

	for i := range t.routes {
		if (t.routes[i].m_mime == nil || t.routes[i].m_mime.Match(mt)) &&
			(t.routes[i].m_ext == nil || t.routes[i].m_ext.Match(ext)) {

			t.log.LogPrintf(DEBUG, "ThumbProcess matched route %d", i)

			// XXX fallbacks?
			return t.routes[i].t.
				doThumbnailing(t.routes[i].p, f, ext, mimeType, fsize, cfg)
		}
	}

	// matched no routes - just fug off
	close_err()
	return
}
