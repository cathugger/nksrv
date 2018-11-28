package webib0

import fp "centpd/lib/httpibfileprovider"

type WebIB0 interface {
	IBProvider          // for web-ish formatting
	fp.HTTPFileProvider // for file serving
	PostProvider        // for posting
}
