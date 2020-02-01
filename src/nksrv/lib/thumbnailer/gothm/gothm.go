package gothm

import (
	"errors"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"strconv"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"nksrv/lib/fstore"
	"nksrv/lib/ftypes"
	. "nksrv/lib/logx"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/thumbnailer/internal/exifhelper"
)

type Config struct {
	MaxWidth, MaxHeight int
	MaxPixels           int
	MaxFileSize         int64
	Filter              imaging.ResampleFilter
	JpegOpts            *jpeg.Options
}

var DefaultConfig = Config{
	MaxWidth:    8192,
	MaxHeight:   8192,
	MaxPixels:   4096 * 4096,
	MaxFileSize: 64 * 1024 * 1024,
	Filter:      imaging.Lanczos,
	JpegOpts:    &jpeg.Options{Quality: 80},
}

func (c Config) BuildThumbnailer(
	fs *fstore.FStore, lx LoggerX) (thumbnailer.Thumbnailer, error) {

	return &GoThumbnailer{cfg: c, fs: fs, log: NewLogToX(lx, "gothm")}, nil
}

type GoThumbnailer struct {
	cfg Config
	fs  *fstore.FStore
	log Logger
}

var errColorFormat = errors.New("unknown color format")

func decodeColor(c string) (color.NRGBA, error) {
	if (len(c) != 4 && len(c) != 7) || c[0] != '#' {
		return color.NRGBA{}, errColorFormat
	}

	i, err := strconv.ParseUint(c[1:], 16, 32)
	if err != nil {
		return color.NRGBA{}, errColorFormat
	}

	if len(c) == 7 {
		return color.NRGBA{
			R: uint8(i >> 16),
			G: uint8(i >> 8),
			B: uint8(i),
			A: 0xFF,
		}, nil
	} else {
		convol := func(x uint8) uint8 { return 0x11 * (0x0F & x) }
		return color.NRGBA{
			R: convol(uint8(i >> 8)),
			G: convol(uint8(i >> 4)),
			B: convol(uint8(i)),
			A: 0xFF,
		}, nil
	}
}

func rotimg(orient int, img *image.NRGBA) *image.NRGBA {
	switch orient {
	case 1:
		// nothing
	case 2:
		img = imaging.FlipV(img)
	case 3:
		img = imaging.Rotate180(img)
	case 4:
		img = imaging.Rotate180(imaging.FlipV(img))
	case 5:
		img = imaging.Rotate270(imaging.FlipV(img))
	case 6:
		img = imaging.Rotate270(img)
	case 7:
		img = imaging.Rotate90(imaging.FlipV(img))
	case 8:
		img = imaging.Rotate90(img)
	}
	return img
}

func (t *GoThumbnailer) ThumbProcess(
	f *os.File, ext, mimeType string, cfg thumbnailer.ThumbConfig) (
	res thumbnailer.ThumbResult, err error) {

	closed := false

	defer func() {
		if !closed {
			f.Close()
		}
	}()

	close_err := func() {
		err = f.Close()
		closed = true
	}

	t.log.LogPrintf(DEBUG, "ThumbProcess ext %q mime %q", ext, mimeType)

	if t.cfg.MaxFileSize > 0 {
		var st os.FileInfo
		st, err = f.Stat()
		if err != nil {
			return
		}
		if st.Size() > t.cfg.MaxFileSize {
			t.log.LogPrintf(DEBUG, "bailing out because filesize limit")
			close_err()
			return
		}
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return
	}

	imgcfg, cfgfmt, ex := image.DecodeConfig(f)
	if ex != nil {
		// bail out on any decoder failure
		t.log.LogPrintf(DEBUG, "bailing out because DecodeConfig err: %v", ex)
		close_err()
		return
	}
	switch cfgfmt {
	case "jpeg", "png", "gif", "webp", "bmp":
		// OK
		t.log.LogPrintf(DEBUG, "detected OK format %q", cfgfmt)
	default:
		// NAK
		t.log.LogPrintf(DEBUG, "detected NAK format %q", cfgfmt)
		close_err()
		return
	}

	// mark this as image & store current config as we know it
	res.FI.Kind = ftypes.FTypeImage
	res.FI.DetectedType = "image/" + cfgfmt

	// seek to start
	_, err = f.Seek(0, 0)
	if err != nil {
		return
	}

	// get orientation
	orient := exifhelper.ExifOrient(f)

	// rotate limits
	imgcfg.Width, imgcfg.Height =
		exifhelper.RotWH(orient, imgcfg.Width, imgcfg.Height)

	res.FI.Attrib = make(map[string]interface{})
	res.FI.Attrib["width"] = imgcfg.Width
	res.FI.Attrib["height"] = imgcfg.Height

	t.log.LogPrintf(
		DEBUG, "after orient size %dx%d", imgcfg.Width, imgcfg.Height)

	if (t.cfg.MaxWidth > 0 && imgcfg.Width > t.cfg.MaxWidth) ||
		(t.cfg.MaxHeight > 0 && imgcfg.Height > t.cfg.MaxHeight) ||
		(t.cfg.MaxPixels > 0 && imgcfg.Width*imgcfg.Height > t.cfg.MaxPixels) {

		// too large, don't do decoding
		t.log.LogPrintf(
			DEBUG, "bailing out because constrained by limits; cfg: %#v", t.cfg)

		close_err()
		return
	}

	// seek to start
	_, err = f.Seek(0, 0)
	if err != nil {
		return
	}

	// XXX golang doesn't care about color profiles at all
	// this means anything non-sRGB is fucked
	// which is sorta unfortunate but oh well.
	// I've failed to find any lib which would help for that
	// and linking to lcms wouldn't be pure go
	// and I aint going to make my own lib for that
	// so I can't really fix this atm.
	oimg, imgfmt, ex := image.Decode(f)
	if ex != nil {
		// bail out on any decoder failure
		t.log.LogPrintf(DEBUG, "bailing out because Decode err: %v", ex)
		close_err()
		return
	}

	close_err()
	if err != nil {
		return
	}

	// width/height
	ow := oimg.Bounds().Dx()
	oh := oimg.Bounds().Dy()
	ow, oh = exifhelper.RotWH(orient, ow, oh)

	cw, ch := exifhelper.RotWH(orient, cfg.Width, cfg.Height)

	// thumbnail
	timg := imaging.Fit(oimg, cw, ch, t.cfg.Filter)

	// after thumbnailing rotate if needed (it costs less there)
	timg = rotimg(orient, timg)

	tsz := timg.Bounds().Size()

	// grayscale
	if cfg.Grayscale {
		timg = imaging.Grayscale(timg)
	}
	// flatten
	if cfg.Color != "" {
		var col color.NRGBA
		col, err = decodeColor(cfg.Color)
		if err != nil {
			return
		}

		bimg := imaging.New(tsz.X, tsz.Y, col)

		timg = imaging.Overlay(bimg, timg, image.Pt(0, 0), 1.0)
	}

	// write

	tf, err := t.fs.NewFile("tmp", "t-", ".jpg")
	if err != nil {
		return
	}
	tfn := tf.Name()
	defer func() {
		if err != nil {
			tf.Close()
			os.Remove(tfn)
		}
	}()

	err = jpeg.Encode(tf, timg, t.cfg.JpegOpts)
	if err != nil {
		return
	}

	err = tf.Close()
	if err != nil {
		return
	}

	// thumbnail info
	res.Width = tsz.X
	res.Height = tsz.Y
	res.DBSuffix = "jpg"
	res.CF.FullTmpName = tfn
	res.CF.Suffix = "jpg"

	// update original img info (it could differ from previous config?)
	res.FI.DetectedType = "image/" + imgfmt // golang devs seem sane so far
	res.FI.Attrib["width"] = ow
	res.FI.Attrib["height"] = oh

	return
}
