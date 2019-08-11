package extthm

import (
	"errors"
	"fmt"
	"image"
	"os"
	"os/exec"

	"nksrv/lib/ftypes"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/thumbnailer/internal/exifhelper"
)

type magickBackend struct {
	t              *ExternalThumbnailer
	binPath        string
	useGM          bool
	forceJPEGLimit bool
}

var errMagickOutputMisunderstod = errors.New("convert output not understod")

func (b *magickBackend) doThumbnailing(
	p tparams, f *os.File, ext, mimeType string, cfg thumbnailer.ThumbConfig) (
	res thumbnailer.ThumbResult, fi thumbnailer.FileInfo, err error) {

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

	var imgcfg image.Config
	var cfgfmt string

	if true /* TODO: alt scanners? */ {
		/*
		 * how this works:
		 * first we query params about image using golang libs
		 * if image params are OK we pass it to imagemagick
		 * we don't need policy files this way
		 * and we already know what type we need to force IM/GM to use
		 * we still need to read output for resulting thumbnail dimensions tho
		 * (anything else would be conceptually unclean)
		 */

		_, err = f.Seek(0, 0)
		if err != nil {
			return
		}

		var ex error
		imgcfg, cfgfmt, ex = image.DecodeConfig(f)
		if ex != nil {
			// bail out on any decoder failure
			close_err()
			return
		}
		switch cfgfmt {
		case "jpeg", "png", "gif", "webp", "bmp":
			// OK
		default:
			// NAK
			close_err()
			return
		}

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

		// mark this as image and store config
		fi.Kind = ftypes.FTypeImage
		fi.DetectedType = "image/" + cfgfmt
		fi.Attrib = make(map[string]interface{})
		fi.Attrib["width"] = imgcfg.Width
		fi.Attrib["height"] = imgcfg.Height

		if (cfgfmt != "jpeg" || b.forceJPEGLimit) &&
			((b.t.cfg.MaxWidth > 0 && imgcfg.Width > b.t.cfg.MaxWidth) ||
				(b.t.cfg.MaxHeight > 0 && imgcfg.Height > b.t.cfg.MaxHeight) ||
				(b.t.cfg.MaxPixels > 0 &&
					imgcfg.Width*imgcfg.Height > b.t.cfg.MaxPixels)) {

			close_err()

			return
		}
	}

	fn := f.Name()

	close_err()
	if err != nil {
		return
	}

	// park file for convert output
	tf, err := b.t.fs.TempFile("t-", ".jpg")
	if err != nil {
		return
	}
	tfn := tf.Name()
	defer func() {
		if err != nil {
			os.Remove(tfn)
		}
	}()
	err = tf.Close()
	if err != nil {
		return
	}

	// at this point we can call IM/GM

	runfile := b.binPath
	useGM := b.useGM
	args := []string{runfile}
	if useGM {
		args = append(args, "convert")
	}

	if cfgfmt == "jpeg" {
		// special jpeg thing
		if !useGM {
			args = append(
				args, "-define",
				fmt.Sprintf("jpeg:size=%dx%d", cfg.Width, cfg.Height))
		} else {
			// GM uses old syntax
			args = append(
				args, "-size", fmt.Sprintf("%dx%d", cfg.Width, cfg.Height))
		}
	}

	// [0] doesn't hurt
	convsrc := cfgfmt + ":" + fn + "[0]"
	args = append(args, convsrc)
	// NOTE: we thumbnail disregarding profile
	// correct would be first converting to linear RGB, but that's bad for perf
	args = append(args, "-thumbnail", fmt.Sprintf("%dx%d", cfg.Width, cfg.Height))
	args = append(args, "-auto-orient")
	if cfg.Color != "" && cfgfmt != "jpeg" && cfgfmt != "bmp" {
		// tbh this should be always set when doing JPEG thumbnails
		args = append(args, "-background", cfg.Color, "-flatten")
	}
	// TODO use profile path?
	// convert to sRGB colorspace if current is different, and strip profiles and other stuff to make smaller
	if !useGM {
		args = append(args, "-colorspace", "sRGB", "-strip")
	} else {
		// gm's sRGB is super weird one so DON'T
		// XXX following would kill non-sRGB profiles
		//args = append(args, "-strip")
	}
	// output to tfn
	args = append(args, tfn)

	cmd := &exec.Cmd{
		Path: runfile,
		Args: args,
	}

	_, ex := cmd.Output()
	if ex != nil {
		if ee, _ := ex.(*exec.ExitError); ee != nil {
			if ee.ProcessState.ExitCode() == 1 {
				// 1 is used for invalid input I think
				// XXX investigate err?
				os.Remove(tfn)
				return
			}
		}
		// XXX should log
		// if file was bad status shouldve been 1
		// otherwise this was unexpected err
		// (file wasn't bad or it was so bad it killed IM/GM)
		err = ex
		return
	}

	res.Width, res.Height =
		calcDecreaseThumbSize(
			imgcfg.Width, imgcfg.Height, cfg.Width, cfg.Height)
	res.FileName = tfn
	res.FileExt = "jpg"

	return
}
