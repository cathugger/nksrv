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

type msgLineFmtCfg struct {
	PreFirstLine     []byte
	PreNonFirstLine  []byte
	PostFinalLine    []byte
	PostNonFinalLine []byte
	FinalNewline     []byte
	NonFinalNewline  []byte
}

func formatmsg(w io.Writer, tr *TmplRenderer, ni *NodeInfo, p *webib0.IBPostInfo) {
	w.Write(tr.m.PreMsg)

	src, last := 0, 0
	greentext := false
	b := p.Message
	blen := len(b)
	n := true
	preline := func() {
		if n {
			if src != 0 {
				w.Write(tr.m.PreFirstLine)
			} else {
				w.Write(tr.m.PreNonFirstLine)
			}
			n = false
		}
	}
	r := 0
	normalfmt := func(end int) {
		for src < end {
			firstch := n
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
				if firstch {
					greentext = true
					// flush
					w.Write(b[last:src])
					src++
					last = src
					// pre-greentext
					w.Write(tr.m.PreQuote)
					// rest of text is normal
					w.Write(htmlGt)
					continue
				}
				esc = htmlGt
			case '\000':
				esc = htmlNull
			case '\n':
				// flush
				w.Write(b[last:src])
				src++
				last = src
				if greentext {
					// terminate greentext
					w.Write(tr.m.PostQuote)
					greentext = false
				}
				// write out post-line stuff
				if src < blen {
					w.Write(tr.m.PostNonFinalLine)
					w.Write(tr.m.NonFinalNewline)
				} else {
					w.Write(tr.m.PostFinalLine)
					w.Write(tr.m.FinalNewline)
				}
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
	rlen := len(p.References)
	for ; r < rlen; r++ {
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
		tr.m.PreRefTmpl.Execute(w, d)
		t.HTMLEscape(w, b[src:rr.End])
		src = int(rr.End)
		last = src
		tr.m.PostRefTmpl.Execute(w, d)
	}
	normalfmt(blen)
	if !n {
		if greentext {
			// terminate greentext
			w.Write(tr.m.PostQuote)
			//greentext = false
		}
		w.Write(tr.m.PostFinalLine)
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
