package mail

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	au "nksrv/lib/asciiutils"
	"nksrv/lib/bufreader"
)

type MessageHead struct {
	H Headers // message headers
	//HSort []string      // header keys sorted in order they appeared
	B *bufreader.BufReader // message body reader
}

func (mh *MessageHead) Close() error {
	if mh.B != nil {
		dropBufReader(mh.B)
		mh.B = nil
	}
	return nil
}

type limitedHeadReader struct {
	R io.Reader // underlying
	N int64     // remaining
}

var errHeadLimitReached = errors.New("read limit reached, head too large")

func (r *limitedHeadReader) Read(b []byte) (n int, err error) {
	if int64(len(b)) > r.N {
		b = b[0:r.N]
	}
	n, err = r.R.Read(b)
	r.N -= int64(n)
	if err == nil && r.N == 0 && n == 0 {
		err = errHeadLimitReached
	}
	return
}

// ReadHeaders reads headers, also returning buffered reader.
// Users should call mh.Close after they used mh.B. If err is returned,
// closing isn't required.
func ReadHeaders(r io.Reader, headlimit int) (mh MessageHead, err error) {
	var lr *limitedHeadReader
	var br *bufreader.BufReader

	if headlimit > 0 {
		lr = &limitedHeadReader{R: r, N: int64(headlimit)}
		br = obtainBufReader(lr)
	} else {
		br = obtainBufReader(r)
	}

	mh.H, err = readHeaders(br)

	// if we get error that means that error has occured before reaching body
	if err == nil {
		if headlimit > 0 {
			if lr.N == 0 && br.QueuedErr() == errHeadLimitReached {
				br.ResetErr()
			}
			br.SetReader(r)
		}
		mh.B = br
	} else {
		dropBufReader(br)
	}

	return
}

func limitedReadHeadersFromExisting(
	cr *bufreader.BufReader, headlimit int) (H Headers, e error) {

	var orig_r io.Reader
	var lr *limitedHeadReader

	if headlimit > 0 {
		// change underlying reader to limit its consumption
		// rough way to do this but should work probably
		queued := len(cr.Buffered()) // take into account already queued data
		if headlimit > queued {
			headlimit -= queued
		} else {
			headlimit = 0
		}
		orig_r = cr.GetReader()
		lr = &limitedHeadReader{R: orig_r, N: int64(headlimit)}
		cr.SetReader(lr)
	}

	H, e = readHeaders(cr)

	if lr != nil {
		// restore original reader
		cr.SetReader(orig_r)
		if lr.N == 0 && cr.QueuedErr() == errHeadLimitReached {
			cr.ResetErr()
		}
	}

	return
}

func ReadHeadersFromExisting(
	cr *bufreader.BufReader, headlimit int) (H Headers, e error) {

	return limitedReadHeadersFromExisting(cr, headlimit)
}

func (mh *MessageHead) ReadHeaders(headlimit int) (H Headers, e error) {
	return limitedReadHeadersFromExisting(mh.B, headlimit)
}

func SkipHeaders(r io.Reader) (mh MessageHead, err error) {
	br := obtainBufReader(r)

	hadNL := true
	for {
		var c byte
		c, err = br.ReadByte()
		if err != nil {
			dropBufReader(br)
			return
		}
		if c == '\n' && hadNL {
			break
		}
		hadNL = c == '\n'
	}
	mh.B = br
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
	h.Reset()

	H = make(Headers)

	var currHeader, origHeader string

	var est int
	est, e = estimateNumHeaders(br)
	//HSort = make([]string, 0, est)
	// one buffer for string slice
	Hbuf := make([]HeaderVal, 0, est)

	var splits []uint32

	finishCurrent := func() error {
		if len(currHeader) != 0 {
			hcont := h.Bytes()
			if !validHeaderContent(hcont) {
				h.Reset()
				return fmt.Errorf("invalid %q header content %#q", currHeader, hcont)
			}
			hval := HeaderVal{HeaderValInner: HeaderValInner{
				V: string(au.TrimWSBytes(hcont)),
				O: origHeader,
				S: splits,
			}}
			splits = []uint32(nil)
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
		return nil
	}

	nextCont := false
	for {
		b := br.Buffered()
		for len(b) != 0 {
			// continuation processing
			currCont := nextCont

			n := bytes.IndexByte(b, '\n')
			var wb []byte
			if n >= 0 {
				if n == 0 || b[n-1] != '\r' {
					wb = b[:n]
				} else {
					wb = b[:n-1]
				}

				//fmt.Printf("!hdr full line %q\n", wb)

				b = b[n+1:]
				br.Discard(n + 1)
				nextCont = false
			} else {
				n := len(b)
				if n == 0 || b[n-1] != '\r' {
					wb = b[:n]
				} else {
					wb = b[:n-1]
				}

				//fmt.Printf("!hdr full unfinished line %q\n", wb)

				br.Discard(n)
				b = nil
				nextCont = true
			}

			if len(wb) == 0 {
				//fmt.Printf("!empty line - end of headers\n")
				// empty line == end of headers
				goto endHeaders
			}

			//fmt.Printf("!currCont = %v\n", currCont)

			// process header line
			if wb[0] != ' ' && wb[0] != '\t' && !currCont {
				// not a continuation
				// finish current, if any
				e = finishCurrent()
				if e != nil {
					break
				}
				// process it
				nn := bytes.IndexByte(wb, ':')
				if nn < 0 {
					// no ':' -- illegal
					e = errMissingColon
					break
				}
				hn := nn
				// strip possible whitespace before ':'
				for hn != 0 && (wb[hn-1] == ' ' || wb[hn-1] == '\t') {
					hn--
				}
				// empty or invalid
				if hn == 0 {
					e = errEmptyHeaderName
					break
				}
				if !ValidHeaderName(wb[:hn]) {
					e = fmt.Errorf("invalid header name: %q", wb[:hn])
					break
				}
				currHeader, origHeader =
					unsafeMapCanonicalOriginalHeaders(wb[:hn])

				nn++
				// skip one space after ':'
				// XXX should we do this for '\t'? probably not.
				/// content is trimmed anyway
				//if nn < len(wb) && wb[nn] == ' ' {
				//	nn++
				//}
				h.Write(wb[nn:])
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
	if e == nil {
		e = finishCurrent()
	} else {
		h.Reset()
	}
	hdrPool.Put(h)
	return
}
