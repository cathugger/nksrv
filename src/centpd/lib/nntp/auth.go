package nntp

type UserPriv struct {
	AllowRead  bool
	AllowWrite bool
}

func ParseUserPriv(s string, up UserPriv) (_ UserPriv, ok bool) {
	for _, c := range s {
		switch c {
		case 'r':
			up.AllowRead = true
		case 'w':
			up.AllowWrite = true
		default:
			return up, false
		}
	}
	return up, true
}

//type NNTPCertFPProvider interface {
//	NNTPUserByFingerprint(cert *x509.Certificate) *User
//	NNTPUserByAnchor(cert *x509.Certificate, anchor string) *User
//}

func (s *NNTPServer) setupClientDefaults(c *ConnState) {
	// TODO
	c.AllowReading = true
	c.AllowPosting = true
}

func (c *ConnState) advertiseAuth() bool {
	// TODO
	return false
}
