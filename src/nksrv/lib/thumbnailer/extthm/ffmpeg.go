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

	if b.fmt == "" {
		// do we actually want to deal with this?
		for _, f := range fmts {
			switch f {
				case "webm", "mp4", "mp3", "ogg", "flac", "wav":
					goto foundfmt
			}
		}
		// didn't match any
		return
	foundfmt:
		// matched
		;
	}

	gotPics := false
	gotPicsID := 0
	gotBadPics := false
	gotVids := 0
	gotBadVids := false
	vidCodec := ""
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
				switch cname {
					// XXX h263, flv1?
					case "h264", "theora", "vp8", "vp9", "av1":
						// OK
						gotVids++
						if gotVids == 1 {
							vidCodec = cname
						} else if vidCodec != cname {
							vidCodec = ""
						}
					default:
						gotBadVids = true
				}
			}
		} else if ctype == "audio" {
			switch cname {
				case "mp3", "vorbis", "opus", "aac", "flac":
					if !gotAudio {
						gotAudio = true
						audCodec = cname
					} else if audCodec != cname {
						// we no longer know
						audCodec = ""
					}
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

	for _, f := range fmts {
		switch f {
			case "webm":
				if gotVids != 0 || gotBadVids {
					mimeType = "video/webm"
				} else if gotAudio || gotBadAudio {
					mimeType = "audio/webm"
				} else {
					// weird, quit
					return
				}

				"mp4", "mp3", "ogg", "flac", "wav":

		}
	}


}
