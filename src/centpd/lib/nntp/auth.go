package nntp

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

func (s *NNTPServer) setupClientDefaults(c *ConnState) {
	// TODO
	c.AllowReading = true
	c.AllowPosting = true
}

func (c *ConnState) advertiseAuth() bool {
	// TODO
	return false
}
