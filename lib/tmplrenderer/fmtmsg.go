package tmplrenderer

import (
	"../webib0"
	"bytes"
	t "text/template"
)

func fmtmsg(p *webib0.IBPostInfo) string {
	var b bytes.Buffer
	t.HTMLEscape(&b, p.Message)
	return string(b.Bytes())
}

func fmtmsgcat(p *webib0.IBThreadCatalogThread) string {
	var b bytes.Buffer
	t.HTMLEscape(&b, p.Message)
	return string(b.Bytes())
}
