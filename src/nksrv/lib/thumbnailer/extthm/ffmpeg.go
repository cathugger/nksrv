package extthm

import (
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"nksrv/lib/app/base/ftypes"
	"nksrv/lib/thumbnailer"
)

type ffmpegSoxBackend struct {
	t *ExternalThumbnailer
	// needs to be set properly or death will await
	// alternatively can be set via p["fmt"]
	fmt string

	ffprobePath string
	ffmpegPath  string
	soxPath     string
}

func (b *ffmpegSoxBackend) init(
	ffprobeBin, ffmpegBin, soxBin string) (err error) {

	b.ffprobePath, err = exec.LookPath(ffprobeBin)
	if err != nil {
		return
	}
	b.ffmpegPath, err = exec.LookPath(ffmpegBin)
	if err != nil {
		return
	}
	b.soxPath, err = exec.LookPath(soxBin)
	if err != nil {
		return
	}
	return
}

// ffprobe json structs for reading infos we're interested in

type ffprobeDispotision struct {
	AttachedPic int `json:"attached_pic"`
}

type ffprobeStream struct {
	CodecType     string  `json:"codec_type"`
	CodecName     string  `json:"codec_name"`
	CodecLongName string  `json:"codec_long_name"`
	Duration      float64 `json:"duration,string"`
	BitRate       int     `json:"bit_rate,string"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	Profile       string  `json:"profile"`
	Level         int     `json:"level"`
	IsAVC         bool    `json:"is_avc,string"`
	AvgFrameRate  string  `json:"avg_frame_rate"`

	Disposition ffprobeDispotision `json:"disposition"`
}

type ffprobeFormat struct {
	FormatName     string  `json:"format_name"`
	FormatLongName string  `json:"format_long_name"`
	Duration       float64 `json:"duration,string"`
	BitRate        int     `json:"bit_rate,string"`
}

type ffprobeMain struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

func (b *ffmpegSoxBackend) doThumbnailing(
	p tparams, f *os.File, ext, mimeType string, fsize int64,
	cfg thumbnailer.ThumbConfig) (
	res thumbnailer.ThumbResult, err error) {

	closed := false

	defer func() {
		if !closed {
			f.Close()
		}
	}()

	closeErr := func() {
		err = f.Close()
		closed = true
	}

	fn := f.Name()

	closeErr()
	if err != nil {
		return
	}

	// we can't read into video/audio files from golang yet
	// so invoke ffprobe with forced format & json output
	runffprobe := b.ffprobePath
	args := []string{
		runffprobe,
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
	}
	if b.fmt != "" {
		args = append(args, "-f", b.fmt)
	} else if p["fmt"] != "" {
		args = append(args, "-f", p["fmt"])
	}
	args = append(args, fn)

	cmd := &exec.Cmd{
		Path: runffprobe,
		Args: args,
	}
	out, ex := cmd.Output()
	if ex != nil {
		// XXX extract info about what happened wrong
		if ee, _ := ex.(*exec.ExitError); ee != nil {
			if ee.ProcessState.ExitCode() == 1 {
				// 1 is used for invalid input I think
				// XXX should maybe do something extra??
				return
			}
		}
		// if it's something else
		err = ex
		return
	}

	var ffproberes ffprobeMain
	err = json.Unmarshal(out, &ffproberes)
	if err != nil {
		return
	}

	fmts := strings.Split(ffproberes.Format.FormatName, ",")
	knownfmt := ""
	for _, f := range fmts {
		switch f {
		case "webm", "mp4", "mp3", "ogg", "flac", "wav":
			knownfmt = f
			goto foundfmt
		}
	}
	// format not known
	return
	///
foundfmt:
	// matched
	;

	gotPics := false
	gotPicsID := 0
	gotBadPics := false
	gotVids := 0
	gotVidsID := 0
	gotBadVids := false
	vidCodec := ""
	gotAudio := false
	gotAudioID := 0
	gotBadAudio := false
	audCodec := ""
	// so okay, what we're dealing with
	for i := range ffproberes.Streams {
		ctype := ffproberes.Streams[i].CodecType
		cname := ffproberes.Streams[i].CodecName
		if ctype == "video" {
			if ffproberes.Streams[i].Disposition.AttachedPic != 0 {
				// picture attachment
				// there may be few of these but we only really care about first one
				if !gotPics {
					// check if we consider format of picture safe
					switch cname {
					case "jpeg", "mjpeg", "png", "gif", "webp", "bmp":
						// yeah OK
						gotPics = true
						gotPicsID = i
					default:
						// mark unknown pics so it'd do conflict with vids anyway
						gotBadPics = true
					}
				}
			} else {
				// video

				// videos having this aint actually valid
				if ffproberes.Streams[i].AvgFrameRate == "0/0" {
					gotBadVids = true
					goto doneVid
				}

				switch {
				// XXX h263, flv1?
				case cname == "h264" && knownfmt == "mp4":
					// gay
					{
						isavc := ffproberes.Streams[i].IsAVC
						prof := ffproberes.Streams[i].Profile
						lvl := ffproberes.Streams[i].Level
						switch {
						// XXX Constrained Baseline is really the same??
						case isavc &&
							(prof == "Baseline" ||
								prof == "Constrained Baseline"):
							cname = fmt.Sprintf("avc1.42E0%02X", lvl)
						case isavc && prof == "Main":
							cname = fmt.Sprintf("avc1.4D40%02X", lvl)
						case isavc && prof == "High":
							cname = fmt.Sprintf("avc1.6400%02X", lvl)
						default:
							cname = ""
						}
						if cname == "" {
							gotBadVids = true
							break
						}
					}

					fallthrough

				case cname == "theora" && knownfmt == "ogg",
					cname == "vp8" && knownfmt != "ogg",
					cname == "vp9" && knownfmt != "ogg":
					/*cname == "av1" && knownfmt != "ogg":*/

					// XXX AV1's format is even gayer than avc1 but don't bother yet
					// https://aomediacodec.github.io/av1-isobmff/#codecsparam

					// OK
					gotVids++
					if gotVids == 1 {
						vidCodec = cname
						gotVidsID = i
					} else if vidCodec != cname {
						vidCodec = ""
					}

				default:
					gotBadVids = true
				}
			doneVid:
			}
		} else if ctype == "audio" {
			switch {
			case cname == "aac" && knownfmt == "mp4":
				cname = "mp4a.40.2" // gay
				fallthrough
			case cname == "mp3",
				cname == "vorbis",
				cname == "opus",
				cname == "flac":

				if !gotAudio {
					gotAudio = true
					audCodec = cname
					gotAudioID = i
				} else if audCodec != cname {
					// we no longer know
					audCodec = ""
				}

			default:
				if strings.HasPrefix(cname, "pcm_") {
					if !gotAudio {
						gotAudio = true
						gotAudioID = i
					}
					audCodec = ""
				} else {
					gotBadAudio = true
				}
			}
		}
		// don't process other types (yet?)
	}

	// OK
	wantcodec := false
	switch knownfmt {
	case "webm":
		if gotBadVids || (gotVids != 0 && gotBadAudio) {
			mimeType = "video/x-matroska"
		} else if gotVids != 0 {
			mimeType = "video/webm"
			wantcodec = true
		} else if gotBadAudio {
			mimeType = "audio/x-matroska"
		} else if gotAudio {
			mimeType = "audio/webm"
			wantcodec = true
		} else {
			// weird, quit
			return
		}

	case "mp4":
		if gotVids != 0 || gotBadVids {
			mimeType = "video/mp4"
			wantcodec = true
		} else if gotAudio || gotBadAudio {
			mimeType = "audio/mp4"
			wantcodec = true
		} else {
			mimeType = "application/mp4"
		}

	case "ogg":
		if gotVids != 0 || gotBadVids {
			mimeType = "video/ogg"
			wantcodec = true
		} else if gotAudio || gotBadAudio {
			mimeType = "audio/ogg"
			wantcodec = true
		} else {
			mimeType = "application/ogg"
		}

	case "mp3":
		mimeType = "audio/mpeg"
	case "flac":
		mimeType = "audio/flac"
	case "wav":
		mimeType = "audio/wave"
	}

	if wantcodec && !gotBadVids && !gotBadAudio &&
		(gotVids == 0 || vidCodec != "") &&
		(!gotAudio || audCodec != "") {

		codecs := ""
		if vidCodec != "" {
			codecs += vidCodec
		}
		if audCodec != "" {
			if codecs != "" {
				codecs += ", "
			}
			codecs += audCodec
		}

		mimeType = mime.FormatMediaType(
			mimeType, map[string]string{"codecs": codecs})
	}

	var tf *os.File
	var tfn string
	parktmp := func(ending string) {
		// park file for convert output
		tf, err = b.t.fs.NewFile("tmp", "t-", ending)
		if err != nil {
			return
		}
		tfn = tf.Name()
		defer func() {
			if err != nil {
				os.Remove(tfn)
			}
		}()
		err = tf.Close()
		if err != nil {
			return
		}
	}

	// so what we gonna do with it?
	res.FI.DetectedType = mimeType

	if gotVids > 1 || gotBadVids {
		// multiple videos... nah
		return
	}

	runffmpeg := b.ffmpegPath

	if gotVids > 0 {
		// VIDEO
		res.FI.Kind = ftypes.FTypeVideo

		res.FI.Attrib = make(map[string]interface{})
		vw := ffproberes.Streams[gotVidsID].Width
		vh := ffproberes.Streams[gotVidsID].Height
		res.FI.Attrib["width"] = vw
		res.FI.Attrib["height"] = vh
		res.FI.Attrib["length"] = ffproberes.Format.Duration

		if (b.t.cfg.MaxWidth > 0 && vw > b.t.cfg.MaxWidth) ||
			(b.t.cfg.MaxHeight > 0 && vh > b.t.cfg.MaxHeight) ||
			(b.t.cfg.MaxPixels > 0 && vw*vh > b.t.cfg.MaxPixels) {

			return
		}

		// NOTICE:
		// there is interesting way to calculate image entropy in
		// SRNd/install_files/plugins/overchan/overchan.py, gen_thumb_from_video
		// it seems ffmpeg binary can't really do it that way though
		// so we do it other way

		avg_fr := ffproberes.Streams[gotVidsID].AvgFrameRate
		avg_fr_s := strings.IndexByte(avg_fr, '/')
		avg_fr_1, _ := strconv.ParseFloat(avg_fr[:avg_fr_s], 64)
		avg_fr_2, _ := strconv.ParseFloat(avg_fr[avg_fr_s+1:], 64)
		vidfps := avg_fr_1 / avg_fr_2

		//b.t.log.LogPrintf(DEBUG, "ffmpeg vid fps(%v)", vidfps)

		allowt := 10 // first 10 secs
		const costyfps = 60
		if vidfps > costyfps {
			// if over FPS limit, scale down secs we take in
			allowt = int((float64(allowt*costyfps) / vidfps) + 0.5)
			if allowt <= 0 {
				allowt = 1
			}
		}

		parktmp(".jpg")
		if err != nil {
			return
		}

		vfilter :=
			fmt.Sprintf(
				"fps=1,scale=%d:%d:force_original_aspect_ratio=decrease,thumbnail",
				cfg.Width, cfg.Height)
		if cfg.Grayscale {
			vfilter += ",hue=s=0"
		}

		args := []string{
			runffmpeg,
			"-v", "error",
			"-an", "-sn", "-dn",
			"-f", knownfmt,
			"-t", strconv.Itoa(allowt),
			"-i", fn,
			"-map", fmt.Sprintf("0:%d", gotVidsID),
			"-vf", vfilter,
			"-frames:v", "1",
			"-qscale:v", "2", // TODO make configurable
			"-y", tfn,
		}
		cmd := &exec.Cmd{
			Path: runffmpeg,
			Args: args,
		}
		_, ex := cmd.Output()
		if ex != nil {
			if ee, _ := ex.(*exec.ExitError); ee != nil {
				if ee.ProcessState.ExitCode() > 0 {
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

		// succeeded
		res.Width, res.Height =
			calcDecreaseThumbSize(vw, vh, cfg.Width, cfg.Height)
		res.DBSuffix = "jpg"
		res.CF.FullTmpName = tfn
		res.CF.Suffix = "jpg"
		return
	}

	// audio file
	if gotAudio {

		// AUDIO
		res.FI.Kind = ftypes.FTypeAudio
		res.FI.Attrib = make(map[string]interface{})
		if ffproberes.Streams[gotAudioID].Duration > 0 {
			res.FI.Attrib["length"] = ffproberes.Streams[gotAudioID].Duration
		} else {
			res.FI.Attrib["length"] = ffproberes.Format.Duration
		}

		if gotPics {
			// XXX
			_ = gotBadPics

			// go is gay
			var aw, ah int
			if cfg.AudioWidth > 0 {
				aw = cfg.AudioWidth
			} else {
				aw = cfg.Width
			}
			if cfg.AudioHeight > 0 {
				ah = cfg.AudioHeight
			} else {
				ah = cfg.Height
			}

			res.FI.Attrib["width"] = aw
			res.FI.Attrib["height"] = ah

			vw := ffproberes.Streams[gotPicsID].Width
			vh := ffproberes.Streams[gotPicsID].Height

			if (b.t.cfg.MaxWidth > 0 && vw > b.t.cfg.MaxWidth) ||
				(b.t.cfg.MaxHeight > 0 && vh > b.t.cfg.MaxHeight) ||
				(b.t.cfg.MaxPixels > 0 && vw*vh > b.t.cfg.MaxPixels) {

				return
			}

			parktmp(".jpg")
			if err != nil {
				return
			}

			vfilter :=
				fmt.Sprintf(
					"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d",
					aw, ah, aw, ah)
			if cfg.Grayscale {
				vfilter += ",hue=s=0"
			}

			args := []string{
				runffmpeg,
				"-v", "error",
				"-an", "-sn", "-dn",
				"-f", knownfmt,
				"-i", fn,
				"-map", fmt.Sprintf("0:%d", gotPicsID),
				"-frames:v", "1",
				"-vf", vfilter,
				"-qscale:v", "2", // TODO make configurable
				"-y", tfn,
			}
			cmd := &exec.Cmd{
				Path: runffmpeg,
				Args: args,
			}
			_, ex := cmd.Output()
			if ex != nil {
				if ee, _ := ex.(*exec.ExitError); ee != nil {
					if ee.ProcessState.ExitCode() > 0 {
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

			// succeeded
			res.Width, res.Height =
				calcDecreaseThumbSize(aw, ah, cfg.Width, cfg.Height)
			res.DBSuffix = "jpg"
			res.CF.FullTmpName = tfn
			res.CF.Suffix = "jpg"
			return
		}

		// ok we've got no pics, if we have bad audio streams don't do anything else
		if gotBadAudio {
			return
		}

		// TODO use sox to thumbnail
		return
	}

	return
}
