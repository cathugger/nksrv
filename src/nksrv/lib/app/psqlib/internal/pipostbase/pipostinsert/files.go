package pipostinsert

import (
	"nksrv/lib/mailib"
	"nksrv/lib/psqlib/internal/pipostbase"
)

type singleFileStuff struct {
	ftype    string
	fsize    int64
	fid      string
	fthumb   string
	forig    string
	fattrib  []byte
	ftattrib []byte
	fextra   []byte
}

func makeSingleFileStuff(fi mailib.FileInfo) singleFileStuff {
	return singleFileStuff{
		ftype:    fi.Type.String(),
		fsize:    fi.Size,
		fid:      fi.ID,
		fthumb:   fi.ThumbField,
		forig:    fi.Original,
		fattrib:  pipostbase.MustMarshal(fi.FileAttrib),
		ftattrib: pipostbase.MustMarshal(fi.ThumbAttrib),
		fextra:   pipostbase.MustMarshal(fi.Extras),
	}
}

type multiFileStuff struct {
	ftypes    []string
	fsizes    []int64
	fids      []string
	fthumbs   []string
	forigs    []string
	fattribs  [][]byte
	ftattribs [][]byte
	fextras   [][]byte
}

func makeMultiFileStuff(fis []mailib.FileInfo) (r multiFileStuff) {
	for _, fi := range fis {
		r.ftypes = append(r.ftypes,
			fi.Type.String())
		r.fsizes = append(r.fsizes,
			fi.Size)
		r.fids = append(r.fids,
			fi.ID)
		r.fthumbs = append(r.fthumbs,
			fi.ThumbField)
		r.forigs = append(r.forigs,
			fi.Original)
		r.fattribs = append(r.fattribs,
			pipostbase.MustMarshal(fi.FileAttrib))
		r.ftattribs = append(r.ftattribs,
			pipostbase.MustMarshal(fi.ThumbAttrib))
		r.fextras = append(r.fextras,
			pipostbase.MustMarshal(fi.Extras))
	}
	return
}
