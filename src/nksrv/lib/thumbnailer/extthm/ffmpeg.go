package extthm

//sect.Add("audio/*", "{{.ffmpeg}} -i {{.infile}} -an -vcodec copy {{.outfile}}")
//sect.Add("video/*", "{{.ffmpeg}} -i {{.infile}} -vf scale=300:200 -vframes 1 {{.outfile}}")

type ffmpegSoxBackend struct {
	t           *ExternalThumbnailer
	fmt         string // needs to be set properly or death will await

	ffprobePath string
	ffmprgPath  string
	soxPath     string
}

// ffprobe json structs for reading infos we're interested in

type ffprobeDispotision struct {
	AttachedPic int `json:"attached_pic"`
}

type ffprobeStream struct {
	CodecType      string  `json:"codec_type"`
	CodecName      string  `json:"codec_name"`
	CodecLongName  string  `json:"codec_long_name"`
	Duration       float64 `json:"duration,string"`
	BitRate        int     `json:"bit_rate,string"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Profile        string  `json:"profile"`
	Level          int     `json:"level"`
	IsAVC          bool    `json:"is_avc,string"`

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

	fn := f.Name()

	close_err()
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
		"-show_format", "-show_streams",
	}
	if b.fmt != "" {
		args = append(args, "-f", b.fmt)
	}
	args = append(args, fn)

	cmd := &exec.Cmd{
		Path: runfile,
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
		err = ee
		return
	}

	var ffproberes ffprobeMain
	err = json.Unmarshal(out, &ffproberes)
	if err != nil {
		return
	}

	fmts := strings.Split(ffproberes.FormatName, ",")
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
				switch {
					// XXX h263, flv1?
					case cname == "h264" && knownfmt == "mp4":
						// gay
						{
							isavc := ffproberes.Streams[i].IsAVC
							prof := ffproberes.Streams[i].Profile
							lvl := ffproberes.Streams[i].Level
							switch {
								case isavc && prof == "Baseline":
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
						cname == "vp9" && knownfmt != "ogg",
						cname == "av1" && knownfmt != "ogg":

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
					} else if audCodec != cname {
						// we no longer know
						audCodec = ""
					}

					gotAudio = true
					audCodec = ""
				default:
					if strings.StartsWith(f, "pcm_") {
						gotAudio = true
					} else {
						gotBadAudio = true
					}
			}
		} else {
			// idk
			//err = fmt.Errorf("unknown codec_type: %q", ctype)
			//return
		}
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

	if wantcodec && !gotBadVideo && !gotBadAudio &&
		(gotVideo == 0 || vidCodec != "") &&
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
			mimeType, map[string]string{"codecs":codecs})
	}

	var tf *os.File
	var tfn string
	parktmp := func(){
		// park file for convert output
		tf, err = b.t.fs.TempFile("t-", ".jpg")
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
	fi.DetectedType = mimeType

	if gotVideo > 1 || gotBadVideo {
		// multiple videos... nah
		return
	}
	if gotVideo > 0 {
		// VIDEO
		fi.Kind = ftypes.FTypeVideo

		fi.Attrib = make(map[string]interface{})
		vw := ffproberes.Streams[gotVidsID].Width
		vh := ffproberes.Streams[gotVidsID].Height
		fi.Attrib["width"]  = vw
		fi.Attrib["height"] = vh
		fi.Attrib["length"] = ffproberes.Format.Duration

		if (b.t.cfg.MaxWidth > 0 && vw > b.t.cfg.MaxWidth) ||
			(b.t.cfg.MaxHeight > 0 && vh > b.t.cfg.MaxHeight) ||
			(b.t.cfg.MaxPixels > 0 && vw*vh > b.t.cfg.MaxPixels) {

			return
		}

		parktmp()
		if err != nil {
			return
		}

		runffmpeg := b.ffmpegPath
		args := []string{
			runffmpeg,
			"-v", "error",
			"-f", knownfmt,
			"-i", fn,
			"-map", fmt.Sprintf("0:%d", gotVidsID),
			"-an", "-sn", "-dn",
			"-vf", fmt.Sprintf(
				"scale=%d:%d:force_original_aspect_ratio=decrease",
				cfg.Width, cfg.Height),
			"-vframes", "1",
			tfn,
		}
		cmd := &exec.Cmd{
			Path: runfile,
			Args: args,
		}
		out, ex := cmd.Output()
		if ex != nil {

		}
	}

}
