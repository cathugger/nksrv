package mail

// common email headers statically allocated to avoid dynamic allocations
// TODO actually analyse which are used and update accordingly
var commonHeaders = map[string]string{
	// overrides
	// RFCs digestion, also observation of actual messages
	"Message-Id":        "Message-ID",
	"Content-Id":        "Content-ID",
	"List-Id":           "List-ID",
	"Mime-Version":      "MIME-Version",
	"Nntp-Posting-Date": "NNTP-Posting-Date",
	"Nntp-Posting-Host": "NNTP-Posting-Host",
	// overchan
	"X-Pubkey-Ed25519":           "X-PubKey-Ed25519",
	"X-Signature-Ed25519-Sha512": "X-Signature-Ed25519-SHA512",
	"X-Frontend-Pubkey":          "X-Frontend-PubKey", // signature below
	"X-Encrypted-Ip":             "X-Encrypted-IP",
	"X-I2p-Desthash":             "X-I2P-DestHash",
}

var commonHeadersList = [...]string{
	// kitchen-sink RFCs and other online sources digestion

	// I really would like to use "cc" and "bcc" as in older RFCs
	// but newer ones have it defined as "Cc" and "Bcc"
	// and that's what is also used by most clients :<

	"Also-Control",
	"Approved",
	"Archive",
	"Article-Names",
	"Article-Updates",
	"Bcc",
	"Bytes",
	"Cancel-Key",
	"Cancel-Lock",
	"Cc",
	"Comments",
	"Content-Description",
	"Content-Disposition",
	"Content-Language",
	"Content-Transfer-Encoding",
	"Content-Type",
	"Control",
	"Date",
	"Date-Received",
	"Distribution",
	"Expires",
	"Face",
	"Followup-To",
	"From",
	"Importance",
	"In-Reply-To",
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
	"See-Also",
	"Sender",
	"Subject",
	"Summary",
	"Supersedes",
	"To",
	"User-Agent",
	"Xref",
	"X-Antivirus",
	"X-Antivirus-Status",
	"X-Complaints-To",
	"X-Complaints-Info",
	"X-Face",
	"X-Mailer",
	"X-Mozilla-News-Host",
	"X-Newsreader",
	"X-Notice",
	"X-Original-Bytes",
	"X-Priority",
	"X-Received",
	"X-Received-Bytes",
	"X-Trace",
	// overchan
	"X-Frontend-Signature", // pubkey above
	"X-Tor-Poster",
	"X-Sage",
}

func AddCommonKey(h string) {
	if len(h) > maxCommonHdrLen {
		panic("maxCommonHdrLen needs adjustment")
	}
	commonHeaders[h] = h
}

func AddCommonKeyOverride(h, o string) {
	if len(h) > maxCommonHdrLen || len(o) > maxCommonHdrLen {
		panic("maxCommonHdrLen needs adjustment")
	}
	commonHeaders[h] = o // override
	commonHeaders[o] = o // self-map
}

func init() {
	// self-map overrides, to allow more efficient lookup
	for _, v := range commonHeaders {
		commonHeaders[v] = v
	}
	// common headers which match their canonical versions
	for _, v := range commonHeadersList {
		commonHeaders[v] = v
	}
}

// does not allocate anything, just returns canonical form if header is common and empty string otherwise
func FindCommonCanonicalKey(s string) string {
	if y, ok := commonHeaders[s]; ok {
		return y
	}

	if len(s) > maxCommonHdrLen {
		return "" // not common
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

func canonicaliseSlice(b []byte) {
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
}

// unsafeMapCanonicalOriginalHeaders maps header name to its
// canonical form, also returning original header form
// if we can't be sure of its canonical form. May modify buffer.
func unsafeMapCanonicalOriginalHeaders(b []byte) (string, string) {
	// fast path: maybe its common header in form we want
	if h, ok := commonHeaders[string(b)]; ok {
		return h, ""
	}
	// save original form
	orig := string(b)
	// canonicalise
	canonicaliseSlice(b)
	// try to use static name again
	if h, ok := commonHeaders[string(b)]; ok {
		// if it works, then we're sure of its canonical form
		return h, ""
	}
	// ohwell nothing we can do, just copy
	can := string(b)
	if can == orig {
		return can, ""
	} else {
		return can, orig
	}
}

func UnsafeCanonicalHeader(b []byte) string {
	// fast path: maybe its common header in form we want
	if h, ok := commonHeaders[string(b)]; ok {
		return h
	}
	// canonicalise
	canonicaliseSlice(b)
	// try to use static name again
	if h, ok := commonHeaders[string(b)]; ok {
		return h
	}
	// ohwell nothing we can do, return unsafe slice
	return unsafeBytesToStr(b)
}
