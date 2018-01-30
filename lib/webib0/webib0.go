package webib0

type WebIB0 interface {
	IBProvider       // for web-ish formatting
	HTTPFileProvider // for file serving
	PostProvider     // for posting
}
