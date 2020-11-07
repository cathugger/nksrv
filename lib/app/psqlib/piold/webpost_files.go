package psqlib

import (
	"os"
	"strings"

	"nksrv/lib/app/mailib"
	ib0 "nksrv/lib/app/webib0"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/mail/form"
	"nksrv/lib/utils/emime"
	ht "nksrv/lib/utils/hashtools"
	au "nksrv/lib/utils/text/asciiutils"
)

var FileFields = ib0.IBWebFormFileFields

func allowedFileName(fname string, slimits *submissionLimits, reply bool) bool {
	if strings.IndexByte(fname, '.') < 0 {
		// we care only about extension anyway so fix that if theres none
		fname = "."
	}
	iffound := slimits.ExtWhitelist
	var list []string
	if !slimits.ExtWhitelist {
		list = slimits.ExtDeny
	} else {
		list = slimits.ExtAllow
	}
	for _, e := range list {
		if matchExtension(fname, e) {
			return iffound
		}
	}
	return !iffound
}

func matchExtension(fn, ext string) bool {
	return len(fn) > len(ext) &&
		au.EndsWithFoldString(fn, ext) &&
		fn[len(fn)-len(ext)-1] == '.'
}

func checkFileLimits(slimits *submissionLimits, reply bool, f form.Form) (err error, c int) {
	var onesz, allsz int64
	for _, fieldname := range FileFields {
		files := f.Files[fieldname]
		c += len(files)
		if c > int(slimits.FileMaxNum) {
			err = errTooMuchFiles(slimits.FileMaxNum)
			return
		}
		for i := range files {
			onesz = files[i].Size
			if slimits.FileMaxSizeSingle > 0 && onesz > slimits.FileMaxSizeSingle {
				err = errTooBigFileSingle(slimits.FileMaxSizeSingle)
				return
			}

			allsz += onesz
			if slimits.FileMaxSizeAll > 0 && allsz > slimits.FileMaxSizeAll {
				err = errTooBigFileAll(slimits.FileMaxSizeAll)
				return
			}

			if !allowedFileName(files[i].FileName, slimits, reply) {
				err = errFileTypeNotAllowed
				return
			}
		}
	}
	if c < int(slimits.FileMinNum) {
		err = errNotEnoughFiles(slimits.FileMinNum)
		return
	}
	return
}

func countRealFiles(FI []mailib.FileInfo) (FC int) {
	for i := range FI {
		if FI[i].Type.Normal() {
			FC++
		}
	}
	return
}

// expects file to be seeked at 0
func generateFileConfig(
	f *os.File, ct string, fi mailib.FileInfo) (
	_ mailib.FileInfo, ext string, err error) {

	hashname, err := ht.MakeFileHash(f)
	if err != nil {
		return
	}

	// prefer info from file name, try figuring out content-type from it
	// if that fails, try looking into content-type, try figure out filename
	// if both fail, just use given type and given filename

	// append extension, if any
	oname := fi.Original

	ext = fu.SafeExt(oname)

	ctype := emime.MIMECanonicalTypeByExtension(ext)
	if ctype == "" && ct != "" {
		mexts, e := emime.MIMEExtensionsByType(ct)
		if e == nil {
			if len(mexts) != 0 {
				ext = mexts[0]
			}
		} else {
			// bad ct
			ct = ""
		}
	}
	if ctype == "" {
		if ct != "" {
			ctype = ct
		} else {
			ctype = "application/octet-stream"
		}
	}

	if len(ext) != 0 {
		ext = emime.MIMEPreferedExtension(ext)
		hashname += "." + ext
	}

	fi.ID = hashname
	fi.ContentType = ctype
	if oname == "" {
		// yeh this is actually possible, work it around in this case
		fi.Original = hashname
	}

	return fi, ext, err
}
