package tmplrenderer

import (
	"io"
	"strings"
	t "text/template"

	"centpd/lib/webib0"
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

func formatmsg(
	w io.Writer, tr *TmplRenderer, ni *NodeInfo,
	boardInfo *webib0.IBBoardInfo, threadInfo *webib0.IBCommonThread,
	p *webib0.IBPostInfo, fullURLs bool, linelimit, charsperline int) (
	err error) {

	_, err = w.Write(tr.m.PreMsg)
	if err != nil {
		return
	}

	lines := 0
	src, last := 0, 0
	greentext := false
	b := p.Message
	blen := len(b)
	n := true               // whether we're at start of new line
	pendingnewline := false // whether there's pending newline to write
	flushnewline := func(final bool) (fe error) {
		if pendingnewline {
			if !final {
				_, fe = w.Write(tr.m.PostNonFinalLine)
				if fe != nil {
					return
				}
				_, fe = w.Write(tr.m.NonFinalNewline)
				if fe != nil {
					return
				}
			} else {
				_, fe = w.Write(tr.m.PostFinalLine)
				if fe != nil {
					return
				}
				_, fe = w.Write(tr.m.FinalNewline)
				if fe != nil {
					return
				}
			}
			pendingnewline = false
		}
		return
	}
	// if we're in for next line, preline checks whether we can write it,
	// if we can it writes preline, else it writes truncation msg.
	preline := func() (_ bool, fe error) {
		if n {
			// truncation
			if linelimit != 0 {
				c := 0
				for _, ch := range string(b[src:]) {
					if ch == '\n' {
						lines++
						c = 0
						break
					}
					if charsperline != 0 && c >= charsperline {
						// TODO break in middle of line
						lines++
						c = 0
					}
					c++
				}
				if c != 0 {
					lines++
				}
				if lines > linelimit {
					d := struct {
						B *webib0.IBBoardInfo
						T *webib0.IBCommonThread
						P *webib0.IBPostInfo
						N *NodeInfo
					}{
						B: boardInfo,
						T: threadInfo,
						P: p,
						N: ni,
					}
					fe = tr.m.TruncationLineTmpl.Execute(w, d)
					return false, fe
				}
			}

			fe = flushnewline(false)
			if fe != nil {
				return
			}

			if src != 0 {
				_, fe = w.Write(tr.m.PreFirstLine)
			} else {
				_, fe = w.Write(tr.m.PreNonFirstLine)
			}
			if fe != nil {
				return
			}
			n = false
		}
		return true, nil
	}
	r := 0
	normalfmt := func(end int) (ok bool, fe error) {
		for src < end {
			firstch := n
			ok, fe = preline()
			if !ok || fe != nil {
				return
			}
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
					_, fe = w.Write(b[last:src])
					if fe != nil {
						return
					}
					src++
					last = src
					// pre-greentext
					_, fe = w.Write(tr.m.PreQuote)
					if fe != nil {
						return
					}
					// rest of text is normal
					_, fe = w.Write(htmlGt)
					if fe != nil {
						return
					}
					continue
				}
				esc = htmlGt
			case '\000':
				esc = htmlNull
			case '\n':
				// flush
				_, fe = w.Write(b[last:src])
				if fe != nil {
					return
				}
				src++
				last = src

				if greentext {
					// terminate greentext
					_, fe = w.Write(tr.m.PostQuote)
					if fe != nil {
						return
					}
					greentext = false
				}

				n = true
				pendingnewline = true
				continue
			case '\r':
				// skip
			default:
				// dont interpret
				src++
				continue
			}
			// flush stuff before replacement
			_, fe = w.Write(b[last:src])
			if fe != nil {
				return
			}
			// write replacement
			_, fe = w.Write(esc)
			if fe != nil {
				return
			}
			// skip some ammount
			src += inc
			// set new mark
			last = src
		}
		// flush
		_, fe = w.Write(b[last:src])
		if fe != nil {
			return
		}
		last = src

		return true, nil
	}

	var cont bool

	rlen := len(p.References)
	for ; r < rlen; r++ {
		rr := &p.References[r]
		if rr.Start > rr.End || rr.End > uint(blen) {
			break
		}

		cont, err = normalfmt(int(rr.Start))
		if err != nil {
			return
		}
		if !cont {
			goto endmsg
		}

		cont, err = preline()
		if err != nil {
			return
		}
		if !cont {
			goto endmsg
		}

		d := struct {
			B *webib0.IBBoardInfo
			T *webib0.IBCommonThread
			P *webib0.IBPostInfo
			R *webib0.IBReference
			F bool
			N *NodeInfo
		}{
			B: boardInfo,
			T: threadInfo,
			P: p,
			R: &rr.IBReference,
			F: fullURLs,
			N: ni,
		}
		err = tr.m.PreRefTmpl.Execute(w, d)
		if err != nil {
			return
		}

		t.HTMLEscape(w, b[src:rr.End])
		src = int(rr.End)
		last = src

		err = tr.m.PostRefTmpl.Execute(w, d)
		if err != nil {
			return
		}
	}

	cont, err = normalfmt(blen)
	if err != nil {
		return
	}
	if !cont {
		goto endmsg
	}

	err = flushnewline(true)
	if err != nil {
		return
	}

	if !n {
		if greentext {
			// terminate greentext
			_, err = w.Write(tr.m.PostQuote)
			if err != nil {
				return
			}
			//greentext = false
		}

		_, err = w.Write(tr.m.PostFinalLine)
		if err != nil {
			return
		}
	}

endmsg:
	_, err = w.Write(tr.m.PostMsg)
	return
}

func fmtmsg(
	tr *TmplRenderer, n *NodeInfo,
	boardInfo *webib0.IBBoardInfo, threadInfo *webib0.IBCommonThread,
	p *webib0.IBPostInfo, fullURLs interface{}, linelimit, charsperline int) (
	_ string, err error) {

	f, _ := t.IsTrue(fullURLs)

	b := &strings.Builder{}
	err = formatmsg(b, tr, n, boardInfo, threadInfo, p,
		f, linelimit, charsperline)
	return b.String(), err
}

func fmtmsgcat(
	tr *TmplRenderer, n *NodeInfo, p *webib0.IBThreadCatalogThread) string {

	b := &strings.Builder{}
	t.HTMLEscape(b, p.Message) // TODO
	return b.String()
}
