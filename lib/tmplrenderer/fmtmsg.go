package tmplrenderer

import (
	"../webib0"
	"strings"
	t "text/template"
)

func fmtmsg(p *webib0.IBPostInfo) string {
	var b strings.Builder
	t.HTMLEscape(&b, p.Message)
	return b.String()
}

func fmtmsgcat(p *webib0.IBThreadCatalogThread) string {
	var b strings.Builder
	t.HTMLEscape(&b, p.Message)
	return b.String()
}
