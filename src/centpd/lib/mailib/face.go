package mailib

import (
	"bytes"
	"image"
	"image/gif"

	"centpd/lib/fstore"
	"centpd/lib/ftypes"
	"centpd/lib/mail"
	"centpd/lib/thumbnailer"
	"centpd/lib/xface"
)

func extractMessageFace(
	H mail.Headers, src *fstore.FStore) (fn string, fi FileInfo, err error) {

	// TODO Face header

	xf := H.GetFirst("X-Face")
	if xf != "" {
		var fimg image.Image
		fimg, err = xface.XFaceStringToImg(xf)
		if err == nil {
			var b bytes.Buffer

			err = gif.Encode(&b, fimg, nil)
			if err != nil {
				panic(err)
			}

			var hash, hashtype string
			var fsize int64
			fn, hash, hashtype, fsize, _, _, err = takeInFile(
				src, thumbnailer.ThumbExec{}, true,
				"gif", "image/gif", &b, true, nil)
			if err != nil {
				return
			}

			fi = FileInfo{
				Type:        ftypes.FTypeFace,
				ContentType: "image/gif",
				Size:        fsize,
				ID:          hash + "-" + hashtype + ".gif",
				Original:    "X-Face",
			}

			return
		}
	}

	return
}
