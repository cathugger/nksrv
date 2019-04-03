package nntpuserpassmap

import (
	"errors"

	"centpd/lib/nntp"
	upn "centpd/lib/userpassnorm"
)

type node struct {
	nntp.UserInfo

	pass string
}

type UserPassMap struct {
	m map[string]*node
}

func NewUserPassMap() UserPassMap {
	return UserPassMap{m: make(map[string]*node)}
}

var _ nntp.UserPassProvider = UserPassMap{}

func (m UserPassMap) Add(ui nntp.UserInfo, pass string) (err error) {
	ui.Name, err = upn.NormaliseUser(ui.Name)
	if err != nil {
		return
	}
	if pass != "" {
		pass, err = upn.NormalisePass(pass)
		if err != nil {
			return
		}
	}
	_, ex := m.m[ui.Name]
	if ex {
		return errors.New("duplicate username")
	}
	m.m[ui.Name] = &node{
		UserInfo: ui,
		pass:     pass,
	}
	return
}

func (m UserPassMap) NNTPUserPassByName(name string) (_ *nntp.UserInfo, _ string) {
	name, err := upn.NormaliseUser(name)
	if err != nil {
		return
	}
	n := m.m[name]
	if n == nil {
		return
	}
	return &n.UserInfo, n.pass
}

func (m UserPassMap) NNTPCheckPass(ch string, rpass nntp.ClientPassword) bool {
	rets, err := upn.NormalisePass(string(rpass))
	return err == nil && ch == rets
}

func (m UserPassMap) NNTPCheckUserPass(name string, rpass nntp.ClientPassword) *nntp.UserInfo {
	name, err := upn.NormaliseUser(name)
	if err != nil {
		return nil
	}
	n := m.m[name]
	if n == nil {
		return nil
	}
	if n.pass == "" {
		return &n.UserInfo
	}
	rpassstr, err := upn.NormalisePass(string(rpass))
	if err != nil || n.pass != rpassstr {
		return nil
	}
	return &n.UserInfo
}
