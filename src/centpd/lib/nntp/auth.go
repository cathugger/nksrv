package nntp

type UserPriv struct {
	AllowRead  bool
	AllowWrite bool
}

func MergeUserPriv(a, b UserPriv) UserPriv {
	return UserPriv{
		AllowRead:  a.AllowRead || b.AllowRead,
		AllowWrite: a.AllowWrite || b.AllowWrite,
	}
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
