package handler

import (
	"net/http"
	"strings"
)

type Method struct {
	// methods[0]=OPTIONS,handlers[0]=fallback,methods[1:n]--handlers[1:n]
	methods  []string
	handlers []http.Handler
}

func NewMethod() *Method {
	return new(Method).Initialize()
}

func (m *Method) Initialize() *Method {
	m.methods = append(m.methods, "OPTIONS")
	m.handlers = append(m.handlers, nil)
	return m
}

func (m *Method) Handle(method string, handler http.Handler) *Method {
	if len(m.methods) < 1 || len(m.handlers) < 1 {
		panic("Method struct is not properly initialized")
	}

	duplicate := func(mm string) bool {
		for _, s := range m.methods {
			if mm == s {
				return true
			}
		}
		return false
	}

	// not going to use non-standard lowercase or mixed-case methods,
	// so this is fine
	um := strings.ToUpper(method)

	if !duplicate(um) {
		m.methods = append(m.methods, um)
		m.handlers = append(m.handlers, handler)
		// we want to include HEAD too, as it's consistent with GET
		if um == "GET" && !duplicate("HEAD") {
			m.methods = append(m.methods, "HEAD")
			m.handlers = append(m.handlers, handler)
		}
	}

	return m
}

func (m *Method) Fallback(handler http.Handler) *Method {
	if len(m.methods) < 1 || len(m.handlers) < 1 {
		panic("Method struct is not properly initialized")
	}
	m.handlers[0] = handler
	return m
}

func (m *Method) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ml := len(m.methods)
	for i := 1; i < ml; i++ {
		if r.Method == m.methods[i] {
			m.handlers[i].ServeHTTP(w, r)
			return
		}
	}
	// handle OPTIONS
	// 1 string alloc (by strings.Join) + whatever Header().Set() allocates
	if r.Method == "OPTIONS" {
		w.Header().Set("Allow", strings.Join(m.methods, ", "))
	} else {
		if len(m.handlers) > 0 && m.handlers[0] != nil {
			m.handlers[0].ServeHTTP(w, r)
		} else {
			http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

var _ http.Handler = (*Method)(nil)
