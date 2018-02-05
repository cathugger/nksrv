package webib0

import fp "../httpibfileprovider"

type WebIB0 interface {
	IBProvider       // for web-ish formatting
	fp.HTTPFileProvider // for file serving
	PostProvider     // for posting
}
