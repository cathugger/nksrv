package mail

import (
	"fmt"
	"io"
)

type PartWriter struct {
	io.Writer
	b string
	s bool
}

func NewPartWriter(w io.Writer, boundary, pre string) (pw *PartWriter) {
	pw = &PartWriter{
		Writer: w,
		b:      fmt.Sprintf("\n--%s--", boundary),
	}
	if pre != "" {
		w.Write(unsafeStrToBytes(pre))
		pw.s = true
	}
	return
}

func (pw *PartWriter) StartNextPart(H Headers) (err error) {
	var bs string
	if pw.s {
		bs = pw.b[:len(pw.b)-2]
	} else {
		bs = pw.b[1 : len(pw.b)-2]
		pw.s = true
	}
	_, err = fmt.Fprintf(pw.Writer, "%s\n", bs)
	if err != nil {
		return
	}
	if H != nil {
		err = WritePartHeaders(pw.Writer, H, false)
		if err != nil {
			return
		}
	}
	_, err = fmt.Fprintf(pw.Writer, "\n")
	return
}

func (pw *PartWriter) FinishParts(post string) (err error) {
	var bs string
	if pw.s {
		bs = pw.b
	} else {
		bs = pw.b[1:]
		pw.s = true
	}
	_, err = fmt.Fprintf(pw.Writer, "%s\n", bs)
	if err != nil {
		return
	}
	if post != "" {
		fmt.Fprintf(pw.Writer, "%s\n", post)
	}
	return
}
