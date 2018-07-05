package mail

import (
	"bytes"
	"io"

	"nekochan/lib/bufreader"
)

type HeaderAcceptor interface {
	EatHeaderName(b []byte)
	EatHeaderValue(b []byte)
	FinishHeader()
}

type MessageHead struct {
	H Headers // message headers
	//HSort []string      // header keys sorted in order they appeared
	B *bufreader.BufReader // message body reader
}

func (mh MessageHead) Close() error {
	if mh.B != nil {
		mh.B.SetReader(nil)
		mh.B.ResetErr()
		bufPool.Put(mh.B)
	}
	return nil
}

func ReadHeaders(r io.Reader, headlimit int64) (mh MessageHead, e error) {
	br := bufPool.Get().(*bufreader.BufReader)
	br.Drop()
	br.ResetErr()

	var lr *io.LimitedReader
	if headlimit > 0 {
		lr = &io.LimitedReader{R: r, N: headlimit}
		br.SetReader(lr)
	} else {
		br.SetReader(r)
	}

	mh.H, e = readHeaders(br)

	if e == nil {
		if headlimit > 0 {
			if lr.N == 0 && br.QueuedErr() == io.EOF {
				br.ResetErr()
			}
			br.SetReader(r)
		}
		mh.B = br
	} else {
		br.SetReader(nil)
		br.ResetErr()
		bufPool.Put(br)
	}

	return
}

func estimateNumHeaders(br *bufreader.BufReader) (n int, e error) {
	br.CompactBuffer()
	_, e = br.FillBufferUpto(0)
	b := br.Buffered()
	cont := 0 // cont -- spare addition incase header line doesn't end with '\n'
	for i, c := range b {
		if c == '\n' {
			if cont == 0 {
				// \n\n or \n without any previous content -- end of headers
				return
			}
			if i+1 < len(b) && (b[i+1] == ' ' || b[i+1] == '\t') {
				// that's just continuation of previous header
				continue
			}
			n++
			cont = 0
		} else {
			cont = 1
		}
	}
	n += cont
	return
}

func readHeaders(br *bufreader.BufReader) (H Headers, e error) {
	h := hdrPool.Get().(*bytes.Buffer)

	H = make(Headers)

	var currHeader string

	var est int
	est, e = estimateNumHeaders(br)
	//HSort = make([]string, 0, est)
	// one buffer for string slice
	Hbuf := make([]HeaderVal, 0, est)

	finishCurrent := func() {
		if len(currHeader) != 0 {
			hval := HeaderVal(h.Bytes())
			if cs, ok := H[currHeader]; ok {
				H[currHeader] = append(cs, hval)
			} else {
				// mark key in HSort array
				//HSort = append(HSort, currHeader)
				// do not include previous values, as in case of reallocation we don't need them
				Hbuf = append(Hbuf[len(Hbuf):], hval)
				// ensure that append will reallocate and not spill into Hbuf by forcing cap to 1
				H[currHeader] = Hbuf[0:1:1]
			}
			currHeader = ""
		}
		h.Reset()
	}

	for {
		b := br.Buffered()
		for len(b) != 0 {
			n := bytes.IndexByte(b, '\n')
			if n < 0 {
				if len(b) >= 2000 {
					// uh oh
					e = errTooLongHeader
				}
				break
			}

			var wb []byte
			if n == 0 || b[n-1] != '\r' {
				wb = b[:n]
			} else {
				wb = b[:n-1]
			}

			//fmt.Printf("!hdr full line>%s\n", wb)

			b = b[n+1:]
			br.Discard(n + 1)

			if len(wb) == 0 {
				//fmt.Printf("!empty line - end of headers\n")
				// empty line == end of headers
				goto endHeaders
			}

			// process header line
			if wb[0] != ' ' && wb[0] != '\t' {
				// not a continuation
				// finish current, if any
				finishCurrent()
				// process it
				n := bytes.IndexByte(wb, ':')
				if n < 0 {
					// no ':' -- illegal
					e = errMissingColon
					break
				}
				hn := n
				// strip possible whitespace before ':'
				for hn != 0 && (wb[hn-1] == ' ' || wb[hn-1] == '\t') {
					hn--
				}
				// empty or invalid
				if hn == 0 || !ValidHeader(wb[:hn]) {
					e = errEmptyHeaderName
					break
				}
				currHeader = mapCanonicalHeader(wb[:hn])

				n++
				// skip one space after ':'
				// XXX should we do this for '\t'?
				if n < len(wb) && wb[n] == ' ' {
					n++
				}
				h.Write(wb[n:])
			} else {
				// a continuation
				if len(currHeader) == 0 {
					// there was no previous header
					e = errInvalidContinuation
					break
				}
				h.Write(wb)
			}
		}
		if e != nil {
			break
		}
		// ensure atleast 2000 bytes space available
		if br.Capacity() < 2000 {
			br.CompactBuffer()
		}
		// pull stuff into buffer
		_, e = br.FillBufferAtleast(1)
	}
endHeaders:
	finishCurrent()
	hdrPool.Put(h)
	return
}
