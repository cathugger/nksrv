package nntp

import (
	"crypto/tls"

	. "centpd/lib/logx"
)

type UserPriv struct {
	AllowReading bool
	AllowPosting bool
}

func MergeUserPriv(a, b UserPriv) UserPriv {
	return UserPriv{
		AllowReading: a.AllowReading || b.AllowReading,
		AllowPosting: a.AllowPosting || b.AllowPosting,
	}
}

func ParseUserPriv(s string, up UserPriv) (_ UserPriv, ok bool) {
	for _, c := range s {
		switch c {
		case 'r':
			up.AllowReading = true
		case 'w':
			up.AllowPosting = true
		default:
			return up, false
		}
	}
	return up, true
}

type UserInfo struct {
	Name string
	Serv string

	UserPriv
}

func (c *ConnState) setupDefaults(rCfg *NNTPServerRunCfg) {
	c.UserPriv = rCfg.DefaultPriv
}

func (c *ConnState) postTLS(rCfg *NNTPServerRunCfg, tlsc *tls.Conn) {
	c.tlsconn = tlsc
	c.UserPriv = MergeUserPriv(c.UserPriv, rCfg.TLSPriv)

	if rCfg.CertFPAutoAuth && !c.authenticated {
		cs := tlsc.ConnectionState()
		if len(cs.PeerCertificates) != 0 {
			ui := rCfg.CertFPProvider.NNTPUserByFingerprint(cs.PeerCertificates[0])
			if ui != nil {
				c.authenticated = true
				c.UserPriv = MergeUserPriv(c.UserPriv, ui.UserPriv)
				c.log.LogPrintf(NOTICE,
					"authenticated using CertFP as name=%q serv=%q", ui.Name, ui.Serv)
			}
		}
	}
}

func (c *ConnState) advertisePlaintextAuth(rCfg *NNTPServerRunCfg) bool {
	return rCfg.UserPassProvider != nil &&
		!c.authenticated &&
		(rCfg.UnsafePass || c.tlsStarted())
}
