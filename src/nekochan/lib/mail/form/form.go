package form

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"os"

	"nekochan/lib/mail"
)

type FormParser struct {
	MaxHeaderBytes    int
	MaxMemory         int
	MaxFields         int
	MaxFileCount      int
	MaxFileSingleSize int64
	MaxFileAllSize    int64
}

var DefaultFormParser = FormParser{
	MaxHeaderBytes: 16 * 1024,
	MaxMemory:      1024 * 1024,
	MaxFields:      1024,
	MaxFileCount:   256,
}

type FileOpener interface {
	OpenFile() (*os.File, error)
}

type File struct {
	F           *os.File
	ContentType string
	FileName    string
}

func (f File) Remove() {
	fn := f.F.Name()
	f.F.Close()
	if fn != "" {
		os.Remove(fn)
	}
}

type Form struct {
	Value map[string][]string
	Files map[string][]File
}

func (f Form) RemoveAll() {
	for k, v := range f.Files {
		for i := range v {
			v[i].Remove()
		}
		delete(f.Files, k)
	}
}

var (
	errFormTooBig    = errors.New("form submission is too big")
	errTooMuchFields = errors.New("form submission contains too much fields")
	errTooMuchFiles  = errors.New("form submission contains too much files")
)

func ParseForm(r io.Reader, boundary string, textfields, filefields []string, fo FileOpener) (Form, error) {
	return DefaultFormParser.ParseForm(r, boundary, textfields, filefields, fo)
}

func (fp *FormParser) ParseForm(r io.Reader, boundary string, textfields, filefields []string, fo FileOpener) (f Form, e error) {
	defer func() {
		if e != nil {
			f.RemoveAll()
		}
	}()
	wantTextField := func(field string) bool {
		for _, v := range textfields {
			if field == v {
				return true
			}
		}
		return false
	}
	wantFileField := func(field string) bool {
		for _, v := range filefields {
			if field == v {
				return true
			}
		}
		return false
	}
	pr := mail.NewPartReader(r, boundary)
	memleft := fp.MaxMemory
	fieldsleft := fp.MaxFields
	buf := bytes.Buffer{}
	numfiles := 0
	var filebytesleft int64
	if fp.MaxFileAllSize > 0 {
		filebytesleft = fp.MaxFileAllSize
	} else {
		filebytesleft = math.MaxInt64
	}
	var n int64
	for {
		e = pr.NextPart()
		if e != nil {
			if e != io.EOF {
				return
			}
			e = nil
			return
		}
		var H mail.Headers
		H, e = pr.ReadHeaders(fp.MaxHeaderBytes)
		if e != nil {
			e = fmt.Errorf("failed reading part headers: %v", e)
			return
		}
		cd := H.GetFirst("Content-Disposition")
		var disp string
		var dispParam map[string]string
		disp, dispParam, e = mime.ParseMediaType(cd)
		name, fname := dispParam["name"], dispParam["filename"]
		if e != nil || disp != "form-data" || name == "" {
			continue
		}
		ct := H.GetFirst("Content-Type")
		if ct == "" && fname == "" {
			// not file
			if !wantTextField(name) {
				// don't need
				continue
			}
			fieldsleft--
			if fieldsleft < 0 {
				e = errTooMuchFields
				return
			}
			n, e = io.CopyN(&buf, pr, int64(memleft)+1)
			if e != nil && e != io.EOF {
				e = fmt.Errorf("failed copying field content: %v", e)
				return
			}
			if n > int64(memleft) {
				// OOHHHH NOW YOU'VE FUCKED UP
				e = errFormTooBig
				return
			}
			memleft -= int(n)
			f.Value[name] = append(f.Value[name], buf.String())
			buf.Reset()
		} else {
			if !wantFileField(name) {
				// don't need lol
				continue
			}
			numfiles++
			if fp.MaxFileCount >= 0 && numfiles > fp.MaxFileCount {
				e = errTooMuchFiles
				return
			}
			fbl := filebytesleft
			if fp.MaxFileSingleSize > 0 && fbl > fp.MaxFileSingleSize {
				fbl = fp.MaxFileSingleSize
			}
			if fbl <= 0 {
				e = errFormTooBig
				return
			}
			var fw *os.File
			fw, e = fo.OpenFile()
			if e != nil {
				e = fmt.Errorf("failed opening file for storage: %v", e)
				return
			}
			killfile := func() {
				fn := fw.Name()
				fw.Close()
				if fn != "" {
					os.Remove(fn)
				}
			}
			n, e = io.CopyN(fw, pr, fbl+1)
			if e != nil && e != io.EOF {
				killfile()
				e = fmt.Errorf("failed copying file: %v", e)
				return
			}
			if n > fbl {
				killfile()
				e = errFormTooBig
				return
			}
			filebytesleft -= n
			fw.Seek(0, 0)
			f.Files[name] = append(f.Files[name], File{
				F:           fw,
				FileName:    fname,
				ContentType: ct,
			})
		}
	}
}
