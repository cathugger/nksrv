package mail

import (
	"bytes"
	"errors"
	"io"
	"sync"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/bufreader"
)

type ArticleReader interface {
	io.Reader
	ReadByte() (byte, error)
	Discard(n int) (int, error)
}

func ValidHeader(h []byte) bool {
	return au.IsPrintableASCIISlice(h, ':')
}

var bufPool = sync.Pool{
	New: func() interface{} {
		// XXX dangerous
		return bufreader.NewBufReaderSize(nil, 4096)
	},
}

var hdrPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
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

var (
	errTooLongHeader       = errors.New("too long header")
	errMissingColon        = errors.New("missing colon in header")
	errEmptyHeaderName     = errors.New("empty header name")
	errInvalidContinuation = errors.New("invalid header continuation")
)

const maxCommonHdrLen = 32

// common email headers statically allocated to avoid dynamic allocations
// TODO actually analyse which are used and update accordingly
var commonHeaders = map[string]string{
	// overrides
	// RFCs digestion
	"Message-Id":   "Message-ID",
	"Content-Id":   "Content-ID",
	"Mime-Version": "MIME-Version",
	// overchan
	"X-Pubkey-Ed25519":           "X-PubKey-Ed25519",
	"X-Signature-Ed25519-Sha512": "X-Signature-Ed25519-SHA512",
	"X-Frontend-Pubkey":          "X-Frontend-PubKey", // signature below
	"X-Encrypted-Ip":             "X-Encrypted-IP",
	"X-I2p-Desthash":             "X-I2P-Desthash",
}

func init() {
	// self-map overrides
	for _, v := range commonHeaders {
		commonHeaders[v] = v
	}
	// common headers which match their canonical versions
	for _, v := range [...]string{
		// kitchen-sink RFCs and other online sources digestion
		"Also-Control",
		"Approved",
		"Archive",
		"Bcc",
		"Bytes",
		"Cc",
		"Comments",
		"Content-Description",
		"Content-Disposition",
		"Content-Language",
		"Content-Length",
		"Content-Transfer-Encoding",
		"Content-Type",
		"Control",
		"Date",
		"Distribution",
		"Expires",
		"Face",
		"Followup-To",
		"From",
		"Injection-Date",
		"Injection-Info",
		"Keywords",
		"Lines",
		"Newsgroups",
		"Organization",
		"Path",
		"Posting-Version",
		"Received",
		"References",
		"Relay-Version",
		"Return-Path",
		"Reply-To",
		"Sender",
		"Subject",
		"Summary",
		"Supersedes",
		"To",
		"In-Reply-To",
		"User-Agent",
		"Xref",
		"X-Antivirus",
		"X-Antivirus-Status",
		"X-Complaints-To",
		"X-Face",
		"X-Mailer",
		"X-Mozilla-News-Host",
		"X-Newsreader",
		"X-Trace",
		// overchan
		"X-Frontend-Signature", // pubkey above
		"X-Tor-Poster",
		"X-Sage",
	} {
		commonHeaders[v] = v
	}
}

type HeaderVal = string
type Headers map[string][]HeaderVal

func (h Headers) GetFirst(x string) HeaderVal {
	if s, ok := h[x]; ok {
		return s[0]
	}
	return ""
}

func (h Headers) Lookup(x string) []HeaderVal {
	if y, ok := commonHeaders[x]; ok {
		return h[y]
	}
	if s, ok := h[x]; ok {
		return s
	}

	var bx [maxCommonHdrLen]byte
	var b []byte
	if len(x) <= maxCommonHdrLen {
		b = bx[:len(x)]
	} else {
		b = make([]byte, len(x))
	}

	upper := true
	for i := 0; i < len(x); i++ {
		c := x[i]
		if upper && c >= 'a' && c <= 'z' {
			c = c - ('a' - 'A')
		}
		if !upper && c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		b[i] = c
		upper = c == '-'
	}
	// dont use commonHeaders there as there's no difference
	return h[string(b)]
}

func FindCommonCanonicalForm(s string) string {
	if len(s) > maxCommonHdrLen {
		return "" // not common
	}

	if y, ok := commonHeaders[s]; ok {
		return y
	}

	var b [maxCommonHdrLen]byte
	upper := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if upper && c >= 'a' && c <= 'z' {
			c = c - ('a' - 'A')
		}
		if !upper && c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		b[i] = c
		upper = c == '-'
	}
	return commonHeaders[string(b[:len(s)])]
}

// XXX can modify underlying storage
func mapCanonicalHeader(b []byte) string {
	// fast path: maybe its common header in form we want
	if h, ok := commonHeaders[string(b)]; ok {
		return h
	}
	// canonicalise
	upper := true
	for i, c := range b {
		if upper && c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
		if !upper && c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
		upper = c == '-'
	}
	// try to use static name again
	if h, ok := commonHeaders[string(b)]; ok {
		return h
	}
	// ohwell nothing we can do, just copy
	return string(b)
}

type MessageHead struct {
	H Headers // message headers
	//HSort []string      // header keys sorted in order they appeared
	B ArticleReader // message body reader
}

func (mh MessageHead) Close() error {
	if mh.B != nil {
		bufPool.Put(mh.B)
	}
	return nil
}

type HeaderAcceptor interface {
	EatHeaderName(b []byte)
	EatHeaderValue(b []byte)
	FinishHeader()
}

func readHeadersInto(r io.Reader, ha HeaderAcceptor, headlimit int64) (B io.Reader, e error) {
	br := bufPool.Get().(*bufreader.BufReader)
	br.Drop()
	br.Reset()
	if headlimit > 0 {
		br.SetReader(&io.LimitedReader{R: r, N: headlimit})
	} else {
		br.SetReader(r)
	}

	hascurrent := false

	finishCurrent := func() {
		if hascurrent {
			ha.FinishHeader()
		}
	}

	for {
		if br.Capacity() < 2000 {
			br.CompactBuffer()
		}
		_, e = br.FillBufferAtleast(1)
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

			b = b[n+1:]
			br.Discard(n + 1)

			if len(wb) == 0 {
				// empty line == end of headers
				B = br
				//e = nil // shallow error, if it's really bad it'll reemerge
				if headlimit > 0 {
					br.Reset()
					br.SetReader(r)
				}
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

				hascurrent = true
				ha.EatHeaderName(wb[:hn])

				n++
				// skip one space after ':'
				// XXX should we do this for '\t'?
				if n < len(wb) && wb[n] == ' ' {
					n++
				}
				ha.EatHeaderValue(wb[n:])
			} else {
				// a continuation
				if !hascurrent {
					// there was no previous header
					e = errInvalidContinuation
					break
				}
				ha.EatHeaderValue(wb)
			}
		}
		if e != nil {
			break
		}
	}
endHeaders:
	finishCurrent()
	return
}

func ReadHeaders(r io.Reader, headlimit int64) (mh MessageHead, e error) {
	br := bufPool.Get().(*bufreader.BufReader)
	br.Drop()
	br.Reset()
	if headlimit > 0 {
		br.SetReader(&io.LimitedReader{R: r, N: headlimit})
	} else {
		br.SetReader(r)
	}
	h := hdrPool.Get().(*bytes.Buffer)

	mh.H = make(Headers)

	var currHeader string

	var est int
	est, e = estimateNumHeaders(br)
	//mh.HSort = make([]string, 0, est)
	// one buffer for string slice
	Hbuf := make([]HeaderVal, 0, est)

	finishCurrent := func() {
		if len(currHeader) != 0 {
			hval := HeaderVal(h.Bytes())
			if cs, ok := mh.H[currHeader]; ok {
				mh.H[currHeader] = append(cs, hval)
			} else {
				// mark key in HSort array
				//mh.HSort = append(mh.HSort, currHeader)
				// do not include previous values, as in case of reallocation we don't need them
				Hbuf = append(Hbuf[len(Hbuf):], hval)
				// ensure that append will reallocate and not spill into Hbuf by forcing cap to 1
				mh.H[currHeader] = Hbuf[0:1:1]
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
				mh.B = br
				//e = nil // shallow error, if it's really bad it'll reemerge
				if headlimit > 0 {
					br.Reset()
					br.SetReader(r)
				}
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
