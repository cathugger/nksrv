package xdialer

import (
	"errors"
	"net"
	"net/url"

	"golang.org/x/net/proxy"
	"golang.org/x/xerrors"
)

type Dialer = proxy.Dialer

// XXX
func XDial(addr string) (d Dialer, proto, host string, err error) {
	for {
		u, e := url.ParseRequestURI(addr)
		if e == nil {
			proto, host = u.Scheme, u.Host
		} else {
			proto, host = "tcp", addr
			u = nil
		}
		if proto == "socks" || proto == "socks5" {
			var a *proxy.Auth
			if u != nil && u.User != nil {
				a = &proxy.Auth{User: u.User.Username()}
				a.Password, _ = u.User.Password()
			}
			d, e = proxy.SOCKS5("tcp", host, a, d)
			if e != nil {
				err = xerrors.Errorf("SOCKS5 error: %w", e)
				return
			}
			addr = u.Path
			if len(addr) != 0 && addr[0] == '/' {
				addr = addr[1:]
			}
		} else {
			// wasn't chain
			break
		}
	}
	if host == "" {
		err = errors.New("no host specified")
		return
	}
	if d == nil {
		d = &net.Dialer{}
	}
	return
}
