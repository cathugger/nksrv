package handler

import (
	"net/http"
	"strings"
)

type hostRoute struct {
	handler    http.Handler
	host, port string
}

type Host struct {
	routes   []hostRoute
	fallback http.Handler
}

func NewHost() *Host {
	return new(Host).Initialize()
}

func (h *Host) Initialize() *Host {
	h.fallback = http.HandlerFunc(internalServerError)
	return h
}

// quick and dirty
// TODO: maybe improve
func splitHostPort(fullhost string) (string, string) {
	if len(fullhost) == 0 || fullhost[0] != '[' {
		// IPv4/hostname
		end := strings.IndexByte(fullhost, ':')
		if end < 0 {
			// no port part
			return fullhost, ""
		}
		// has something in port part. or not.
		return fullhost[:end], fullhost[end+1:]
	} else {
		// IPv6
		end := strings.IndexByte(fullhost[1:], ']')
		if end < 0 {
			// ???
			return fullhost, ""
		}
		if end+1 >= len(fullhost) || fullhost[end+1] != ':' {
			// has no or fucked up port part
			return fullhost[1:end], ""
		}
		// has something in port part. or not.
		return fullhost[1:end], fullhost[end+2:]
	}
}

func (h *Host) Handle(host, port string, handler http.Handler) *Host {
	r := hostRoute{
		handler: handler,
		host:    host,
		port:    port,
	}
	h.routes = append(h.routes, r)
	return h
}

func (h *Host) Fallback(handler http.Handler) *Host {
	h.fallback = handler
	return h
}

func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, port := splitHostPort(r.Host)
	for i := range h.routes {
		if h.routes[i].host == "*" || strings.EqualFold(h.routes[i].host, host) {
			if h.routes[i].port == "*" || h.routes[i].port == port {
				h.routes[i].handler.ServeHTTP(w, r)
				return
			}
		}
	}
	h.fallback.ServeHTTP(w, r)
}

var _ http.Handler = (*Host)(nil)

// nyu
