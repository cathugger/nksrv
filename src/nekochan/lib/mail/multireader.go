package mail

import (
	"bytes"
	"fmt"
	"io"

	"nekochan/lib/bufreader"
)

type PartReader struct {
	br               *bufreader.BufReader
	dashBoundaryDash []byte
	dashBoundary     []byte
	nlDashBoundary   []byte
	nl               []byte
	partsRead        int
}

func NewPartReader(r io.Reader, boundary string) *PartReader {
	b := []byte("\r\n--" + boundary + "--")
	return &PartReader{
		br:               bufreader.NewBufReaderSize(r, 4096), /* TODO use pool */
		dashBoundaryDash: b[2:],
		dashBoundary:     b[2 : len(b)-2],
		nlDashBoundary:   b[:len(b)-2],
		nl:               b[:2],
	}
}

// returns nil incase next part can be read
// returns io.EOF if terminated
// returns other error on read problem
func (pr *PartReader) NextPart() (e error) {
	// TODO terminate current
	br := pr.br
	truncated := false
	expectNewPart := false
	for {
		b := br.Buffered()
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			// check if we can read more
			if br.Capacity() == 0 {
				// cant read more, try compact
				if br.Size() > len(b) {
					// do compaction
					br.CompactBuffer()
				} else {
					// cant compact, its just too big -- dont process it
					br.Discard(len(b))
					truncated = true
				}
			}
			_, e = br.FillBufferAtleast(1)
			// check error. if error that means buffer don't have new data
			if e != nil {
				// maybe we have reached ending without [\r]\n?
				// if not, EOF is unexpected
				if e == io.EOF && (truncated || !pr.checkPartEnd(br.Buffered())) {
					e = io.ErrUnexpectedEOF
				}
				return
			}
			continue
		}

		line := b[:i+1]
		br.Discard(i + 1)
		// we have line of some sort, check if its boundary
		if !truncated && bytes.HasPrefix(line, pr.dashBoundary) {
			// some sort of boundary maybe
			ending := false
			blen := len(pr.dashBoundary)
			if len(line) <= blen+2 || line[blen] != '-' || line[blen+1] != '-' {
				line = line[blen:]
			} else {
				line = line[blen+2:]
				ending = true
			}
			line = skipWS(line)
			if pr.partsRead == 0 && len(line) == 1 && line[0] == '\n' {
				// adopt to \n
				pr.nl = pr.nl[1:]
				pr.nlDashBoundary = pr.nlDashBoundary[1:]
			}
			if bytes.Equal(line, pr.nl) {
				if !ending {
					pr.partsRead++
					return
				} else {
					return io.EOF
				}
			}
		}
		if expectNewPart {
			return fmt.Errorf("was expecting new part, got line %q", line)
		}
		if pr.partsRead == 0 {
			// skip everything before first part
			continue
		}
		if truncated || bytes.Equal(line, pr.nl) {
			// newline after last part just before dashBoundary
			expectNewPart = true
			continue
		}
		return fmt.Errorf("truncated line or unexpected line %q", line)
	}
}

func skipWS(b []byte) []byte {
	for len(b) != 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	return b
}

func (pr *PartReader) checkPartEnd(line []byte) bool {
	if !bytes.HasPrefix(line, pr.dashBoundaryDash) {
		return false
	}
	line = line[len(pr.dashBoundaryDash):]
	line = skipWS(line)
	return len(line) == 0
}

func (pr *PartReader) Read(b []byte) (n int, e error) {
	for {

	}
}
