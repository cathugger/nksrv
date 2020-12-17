package mailib

import (
	"bytes"
	"encoding/base64"
	"image/gif"
	"image/png"
	"strings"

	"nksrv/lib/app/base/ftypes"
	"nksrv/lib/mail"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/utils/fs/fstore"
	"nksrv/lib/utils/xface"
)

var stripSPHT = strings.NewReplacer(" ", "", "\t", "")

func extractMessageFace(
	H mail.HeaderMap, src *fstore.FStore) (fn string, fi FileInfo, err error) {

	// try Face header first
	fh := H.GetFirst("Face")
	if fh != "" {
		fh = stripSPHT.Replace(fh)
		pngb := make([]byte, base64.StdEncoding.DecodedLen(len(fh)))
		n, ex := base64.StdEncoding.Decode(pngb, unsafeStrToBytes(fh))
		if ex == nil && n > 0 {
			pngb = pngb[:n]
			pngr := bytes.NewReader(pngb)
			// safety check that it can't be harmful to us or clients
			cfg, ex := png.DecodeConfig(pngr)
			if ex == nil && cfg.Width == 48 && cfg.Height == 48 {
				// sanity check that image isn't corrupt
				pngr.Reset(pngb)
				_, ex = png.Decode(pngr)
				if ex == nil {

					// image in buffer suits our needs :>

					var hashname string
					var fsize int64
					pngr.Reset(pngb)
					fn, hashname, fsize, _, err =
						takeInFile(
							src, thumbnailer.ThumbExec{}, true,
							"png", "image/png", pngr, true, nil)
					if err != nil {
						return
					}

					fi = FileInfo{
						Type:        ftypes.FTypeFace,
						ContentType: "image/png",
						Size:        fsize,
						ID:          hashname + ".png",
						Original:    "Face",
					}

					return
				}
			}
		}
	}

	// X-Face header next
	xfh := H.GetFirst("X-Face")
	if xfh != "" {
		fimg, ex := xface.XFaceStringToImg(xfh)
		if ex == nil {
			var b bytes.Buffer

			ex = gif.Encode(&b, fimg, nil)
			if ex != nil {
				panic(ex)
			}

			var hashname string
			var fsize int64
			fn, hashname, fsize, _, err =
				takeInFile(
					src, thumbnailer.ThumbExec{}, true,
					"gif", "image/gif", &b, true, nil)
			if err != nil {
				return
			}

			fi = FileInfo{
				Type:        ftypes.FTypeFace,
				ContentType: "image/gif",
				Size:        fsize,
				ID:          hashname + ".gif",
				Original:    "X-Face",
			}

			return
		}
	}

	return
}
