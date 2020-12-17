package mail

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	au "nksrv/lib/utils/text/asciiutils"
	"nksrv/lib/utils/text/bufreader"
)

type MessageHead struct {
	H HeaderMap // message headers
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

	mh.H, err = readHeaderMap(br)

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
	cr *bufreader.BufReader, headlimit int) (H HeaderMap, e error) {

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

	H, e = readHeaderMap(cr)

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
	cr *bufreader.BufReader, headlimit int) (H HeaderMap, e error) {

	return limitedReadHeadersFromExisting(cr, headlimit)
}

func (mh *MessageHead) ReadHeaders(headlimit int) (H HeaderMap, e error) {
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

type readHeaderFunc = func(k, o, v string, s HeaderValSplitList)

func readHeaderMap(br *bufreader.BufReader) (H HeaderMap, e error) {

	H = make(HeaderMap)

	est, e := estimateNumHeaders(br)

	Hbuf := make([]HeaderMapVal, 0, est)

	f := func(k, o, v string, s HeaderValSplitList) {
		hval := HeaderMapVal{HeaderMapValInner: HeaderMapValInner{
			V: v,
			O: o,
			S: s,
		}}
		if cs, ok := H[k]; ok {
			H[k] = append(cs, hval)
		} else {
			// do not include previous values, as in case of reallocation we don't need them
			Hbuf = append(Hbuf[len(Hbuf):], hval)
			// ensure that append will reallocate and not spill into Hbuf by forcing cap to 1
			H[k] = Hbuf[0:1:1]
		}
	}

	e = readHeaderIntoFunc(br, f)

	return
}

func errInvalidHeaderContent(k string, v []byte) error {
	return fmt.Errorf("invalid %q header content %#q", k, v)
}

func errInvalidHeaderName(k []byte) error {
	return fmt.Errorf("invalid header name: %#q", k)
}

var errEmptyFold = errors.New("empty folding lines aren't allowed")

func readHeaderIntoFunc(br *bufreader.BufReader, rhf readHeaderFunc) (e error) {
	h := hdrPool.Get().(*bytes.Buffer)
	h.Reset()

	var currHeader, origHeader string
	var splits HeaderValSplitList

	// |-------------------|------|
	// 0                   s
	var line []byte   // CURRENT full line
	var start int     // begining of current line's logical fragment
	var contStart int // begining of actual content of whole defrag'd line
	var lastStart int // for fragment counting

	finishCurrent := func() error {

		if len(currHeader) != 0 {

			//fmt.Printf("!hdr finishing current %q\n", currHeader)

			hcont := h.Bytes()[contStart:start]
			if !validHeaderContent(hcont) {
				h.Reset()
				return errInvalidHeaderContent(currHeader, hcont)
			}

			rhf(currHeader, origHeader,
				string(au.TrimWSBytes(hcont)), splits)

			splits = HeaderValSplitList(nil)
			currHeader = ""
		}

		h.Reset()
		start = 0
		lastStart = 0

		return nil
	}

	lastWasFrag := false
	currFrag := false
	for {
		b := br.Buffered()
		for len(b) != 0 {
			// wb is currently usable slice
			var wb []byte
			lastWasFrag = currFrag

			n := bytes.IndexByte(b, '\n')
			currFrag = n < 0
			if !currFrag {
				// found newline - this line will be complete
				wb = b[:n] // do not include LF
				//fmt.Printf("!hdr full line %q\n", wb)
				b = b[n+1:]
				br.Discard(n + 1)
			} else {
				// no newline yet - take in what we can
				wb = b
				//fmt.Printf("!hdr full unfinished line %q\n", wb)
				br.Discard(len(b))
				b = nil
			}

			// we can already know at this point if next completed line
			// is going to be logical fragment or not
			if !lastWasFrag && len(wb) != 0 && wb[0] != ' ' && wb[0] != '\t' {
				// finish current, if any
				e = finishCurrent()
				if e != nil {
					break
				}
			}

			// write it out
			h.Write(wb)

			//fmt.Printf("!hdr wrote chunk %q\n", wb)

			// drain until we have completed this line
			if currFrag {
				continue
			}

			// take current logical fragment
			line = h.Bytes()[start:]
			//fmt.Printf("!hdr got full line[%d] %q\n", start, line)
			// LF was already skipped
			// have CR? discard that too
			if len(line) != 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
				h.Truncate(start + len(line))
			}

			if len(line) == 0 {
				// empty line terminates headers
				goto endHeaders
			}

			// process header line

			if line[0] != ' ' && line[0] != '\t' {
				// not logical continuation, just normal line
				//fmt.Printf("!hdr line is NOT continuation\n")

				// find :
				nn := bytes.IndexByte(line, ':')
				if nn < 0 {
					// no ':' -- illegal
					e = errMissingColon
					break
				}
				hn := nn

				// strip possible whitespace before ':'
				for hn != 0 && (line[hn-1] == ' ' || line[hn-1] == '\t') {
					hn--
				}

				// empty or invalid
				if hn == 0 {
					e = errEmptyHeaderName
					break
				}
				if !ValidHeaderName(line[:hn]) {
					e = errInvalidHeaderName(line[:hn])
					break
				}

				// get proper header string
				currHeader, origHeader =
					unsafeMapCanonicalOriginalHeaders(line[:hn])

				//fmt.Printf("!hdr header name is %q\n", currHeader)

				nn++ // step over ':'

				// trim before actual text
				for nn < len(line) && (line[nn] == ' ' || line[nn] == '\t') {
					nn++
				}

				// mark actual content start
				contStart = nn
				lastStart = contStart

				// mark start of the next line
				start = h.Len()

			} else {
				// logical continuation

				if len(currHeader) == 0 {
					e = errInvalidContinuation
					break
				}
				if start-lastStart == 0 || len(au.TrimLeftWSBytes(line)) == 0 {
					// last fragment was empty or this one is
					// it would be a problem because of how it interacts with trimming
					e = errEmptyFold
					break
				}

				//fmt.Printf("!hdr line is continuation\n")

				splits = append(splits, HeaderValSplit(start-lastStart))
				lastStart = start

				// mark start
				start = h.Len()
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
