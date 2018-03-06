package filelogger

import (
	"bufio"
	"bytes"
	"io"
)

var newline = []byte{'\n'}

var _ io.Writer = (*splitter)(nil)

type splitter struct {
	w *bufio.Writer
	p bytes.Buffer
	n bool
}

func (s *splitter) reset() {
	s.n = true
	s.p.Reset()
}

func (s *splitter) Write(b []byte) (n int, err error) {
	l := 0
	p := s.p.Bytes()
	for i := range b {
		if s.n {
			s.w.Write(p)
			s.n = false
		}
		if b[i] == '\n' {
			s.w.Write(b[l : i+1])
			l = i + 1
			s.n = true
		}
	}
	if l < len(b) {
		s.w.Write(b[l:])
	}
	return len(b), nil
}

func (s *splitter) finish() {
	if !s.n {
		s.w.Write(newline)
		// XXX `s.n = true` is intentionally left out
	}
	s.w.Flush()
}
