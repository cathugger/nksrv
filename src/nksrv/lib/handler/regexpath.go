package handler

import (
	"context"
	"net/http"
	re "regexp"
	"strings"
)

/*
 * regex paths are declared with syntax: /whatever/{{varname[:regex]}}/whateverelse
 * varname is not optional
 * by default regex is `[^/]+`
 * whole expression is prepended with ^ and appended with $
 * it can contain regex inside, but it must not contain captures
 * capturing regex itself must not have captures on its own aswell
 * non-capturing group is specified like this: (?:blablabla)
 */

/*
 * transformation we need to do:
 * detect {{}}, extract name from them, extract optional regex from them
 * replace {{}} with capturing regex, store name and position
 * {{:whatever}} is escape
 */

type regexPathRoute struct {
	handler  http.Handler
	varnames []string
	pattern  *re.Regexp
	strip    bool
}

type RegexPath struct {
	routes   []regexPathRoute
	fallback http.Handler
}

var capregex = re.MustCompile("\\{\\{..*?\\}\\}")

func NewRegexPath() *RegexPath {
	return new(RegexPath).Initialize()
}

func (p *RegexPath) Initialize() *RegexPath {
	p.fallback = http.HandlerFunc(badRequest)
	return p
}

func (p *RegexPath) Handle(pathexp string, strip bool, handler http.Handler) *RegexPath {
	r := regexPathRoute{handler: handler, strip: strip}

	pathexp = capregex.ReplaceAllStringFunc(pathexp, func(capture string) string {
		capture = capture[2 : len(capture)-2]
		var exprcap string
		expression := "[^/]+"
		if i := strings.IndexByte(capture, ':'); i >= 0 {
			exprcap = capture[i+1:]
			expression = exprcap
			capture = capture[:i]
		}
		if len(capture) > 0 {
			r.varnames = append(r.varnames, capture)
			if len(expression) > 0 && expression[0] == '^' {
				expression = expression[1:]
			}
			if len(expression) > 0 && expression[len(expression)-1] == '$' &&
				(len(expression) < 2 ||
					expression[len(expression)-2] != '\\') {
				expression = expression[:len(expression)-1]
			}
			return "(" + expression + ")"
		} else {
			return "{{" + exprcap + "}}"
		}
	})
	if !strip {
		pathexp = "^" + pathexp + "$"
	} else {
		pathexp = "^" + pathexp + "(/.*)$"
	}
	r.pattern = re.MustCompile(pathexp)

	p.routes = append(p.routes, r)

	return p
}

func (p *RegexPath) Fallback(handler http.Handler) *RegexPath {
	p.fallback = handler
	return p
}

func (p *RegexPath) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for i := range p.routes {
		if m := p.routes[i].pattern.FindStringSubmatch(r.URL.Path); m != nil {
			numvars := len(p.routes[i].varnames)
			expected := numvars + 1
			if p.routes[i].strip {
				expected++
			}
			if len(m) != expected {
				// we have no idea which submatch refers to which variable
				panic("regexpath/ServeHTTP: unexpected matched expression count")
			}
			if numvars > 0 {
				ctx := r.Context()
				for j, vn := range p.routes[i].varnames {
					// this place will probably make quite some garbage
					ctx = context.WithValue(ctx, vn, m[j+1])
				}
				// this too
				r = r.WithContext(ctx)
			}
			if p.routes[i].strip {
				// this is destructive
				r.URL.Path = m[numvars+1]
			}
			p.routes[i].handler.ServeHTTP(w, r)
			return
		}
	}
	p.fallback.ServeHTTP(w, r)
}

var _ http.Handler = (*RegexPath)(nil)

// Puchico a cute
