package form

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"os"
	"strings"

	"centpd/lib/mail"
)

type ParserParams struct {
	MaxHeaderBytes    int
	MaxMemory         int
	MaxFields         int
	MaxFileCount      int
	MaxFileSingleSize int64
	MaxFileAllSize    int64
}

var DefaultParserParams = ParserParams{
	MaxHeaderBytes: 16 * 1024,
	MaxMemory:      1024 * 1024,
	MaxFields:      1024,
	MaxFileCount:   256,
}

type FileOpener interface {
	OpenFile() (*os.File, error)
}

type File struct {
	F           *os.File // File is seeked to 0 position
	ContentType string
	FileName    string // Windows and UNIX paths are stripped
	Size        int64
}

func (f File) Remove() {
	fn := f.F.Name()
	f.F.Close()
	if fn != "" {
		os.Remove(fn)
	}
}

type Form struct {
	Values map[string][]string
	Files  map[string][]File
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

func ParseForm(
	r io.Reader, boundary string, textfields, filefields []string,
	fo FileOpener) (Form, error) {

	return DefaultParserParams.ParseForm(
		r, boundary, textfields, filefields, fo)
}

func (fp *ParserParams) ParseForm(
	r io.Reader, boundary string, textfields, filefields []string,
	fo FileOpener) (f Form, e error) {

	defer func() {
		if e != nil {
			f.RemoveAll()
		}
	}()

	f.Values = make(map[string][]string)
	f.Files = make(map[string][]File)

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
		//fmt.Fprintf(os.Stderr, "XXX before NextPart()\n")
		e = pr.NextPart()
		//fmt.Fprintf(os.Stderr, "XXX after NextPart() before ReadHeaders()\n")
		if e != nil {
			if e != io.EOF {
				return
			}
			e = nil
			return
		}
		var H mail.Headers
		H, e = pr.ReadHeaders(fp.MaxHeaderBytes)
		//fmt.Fprintf(os.Stderr, "XXX after ReadHeaders()\n")
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
			// XXX log error? or maybe return error?
			continue
		}
		//fmt.Fprintf(os.Stderr, "XXX form params: name(%q) filename(%q)\n", name, fname)
		if fname == "" {
			//fmt.Fprintf(os.Stderr, "XXX form part is field\n")
			// not file
			if !wantTextField(name) {
				// don't need
				//fmt.Fprintf(os.Stderr, "XXX this field isn't needed\n")
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
			f.Values[name] = append(f.Values[name], buf.String())
			buf.Reset()
		} else {
			//fmt.Fprintf(os.Stderr, "XXX form part is file\n")
			if !wantFileField(name) {
				// don't need lol
				//fmt.Fprintf(os.Stderr, "XXX this file isn't needed\n")
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
			//fmt.Fprintf(os.Stderr, "XXX before CopyN()\n")
			n, e = io.CopyN(fw, pr, fbl+1)
			//fmt.Fprintf(os.Stderr, "XXX after CopyN()\n")
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
			_, e = fw.Seek(0, 0)
			if e != nil {
				killfile()
				e = fmt.Errorf("failed seeking file: %v", e)
				return
			}

			// users will only need this part
			if i := strings.LastIndexAny(fname, "/\\"); i >= 0 {
				fname = fname[i+1:]
			}

			f.Files[name] = append(f.Files[name], File{
				F:           fw,
				FileName:    fname,
				ContentType: H.GetFirst("Content-Type"),
				Size:        n,
			})
		}
	}
}
