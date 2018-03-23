package tmplrenderer

import (
	"../webib0"
	"io"
	"strings"
	t "text/template"
)

// HTML escapes
var (
	htmlQuot = []byte("&#34;") // shorter than "&quot;"
	htmlApos = []byte("&#39;") // shorter than "&apos;" and apos was not in HTML until HTML5
	htmlAmp  = []byte("&amp;")
	htmlLt   = []byte("&lt;")
	htmlGt   = []byte("&gt;")
	htmlNull = []byte("\uFFFD")
)

// we need to have template rendering customizable
// current idea: pre-message, pre-line, post-line, post-message
// issue: message ending should have newline? or not?
// solution: separate newline tag
// things may or may not end with newline. this is completely fine

// hard: we need to have links customizable
// we cant be sure what kind of router we will be behind
// * use golang text/template, or fasttemplate
// fasttemplate cannot do conditions, so would complicate things
// we would need 3 or more templates: for dead link, for boards,
// for remote threads, for current thread, for threads in current board...
// either that or offload forming of link to somewhere else
// maybe routing package itself
// that's bit too much for now
// so I'll just do text/template

func formatmsg(w io.Writer, tr *TmplRenderer, ni *NodeInfo, p *webib0.IBPostInfo) {
	w.Write(tr.m.PreMsg)

	src, last := 0, 0
	//var tags []uint
	b := p.Message
	blen := len(b)
	n := true
	preline := func() {
		if n {
			if src != 0 {
				w.Write(tr.m.FirstPreLine)
			} else {
				w.Write(tr.m.NextPreLine)
			}
			n = false
		}
	}
	normalfmt := func(end int) {
		for src < end {
			//firstch := n
			preline()
			c := b[src]
			inc := 1 // default ammout to skip is one character
			var esc []byte
			switch c {
			case '"':
				esc = htmlQuot
			case '\'':
				esc = htmlApos
			case '&':
				esc = htmlAmp
			case '<':
				esc = htmlLt
			case '>':
				esc = htmlGt
			case '\000':
				esc = htmlNull
			case '\n':
				// flush
				w.Write(b[last:src])
				src++
				last = src
				// write out post-line stuff
				w.Write(tr.m.PostLine)
				w.Write(tr.m.Newline)
				n = true
				continue
			case '\r':
				// skip
			default:
				// dont interpret
				src++
				continue
			}
			// flush stuff before replacement
			w.Write(b[last:src])
			// write replacement
			w.Write(esc)
			// skip some ammount
			src += inc
			// set new mark
			last = src
		}
		// flush
		w.Write(b[last:src])
		last = src
	}
	for r := range p.References {
		rr := &p.References[r]
		if rr.Start > rr.End || rr.End > uint(blen) {
			break
		}
		normalfmt(int(rr.Start))
		preline()
		d := struct {
			D *webib0.IBReference
			N *NodeInfo
		}{
			D: &rr.IBReference,
			N: ni,
		}
		tr.m.preRefTmpl.Execute(w, d)
		t.HTMLEscape(w, b[src:rr.End])
		src = int(rr.End)
		last = src
		tr.m.postRefTmpl.Execute(w, d)
	}
	normalfmt(blen)
	if !n {
		w.Write(tr.m.PostLine)
	}

	w.Write(tr.m.PostMsg)
}

func fmtmsg(tr *TmplRenderer, n *NodeInfo, p *webib0.IBPostInfo) string {
	var b strings.Builder
	formatmsg(&b, tr, n, p)
	return b.String()
}

func fmtmsgcat(tr *TmplRenderer, n *NodeInfo, p *webib0.IBThreadCatalogThread) string {
	var b strings.Builder
	t.HTMLEscape(&b, p.Message) // TODO
	return b.String()
}
