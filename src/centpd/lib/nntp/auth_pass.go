package nntp

type ClientPassword string

type UserPassProvider interface {
	NNTPUserPassByName(name string) (*UserInfo, string)
	NNTPCheckPass(ch string, rpass ClientPassword) bool
	NNTPCheckUserPass(username string, rpass ClientPassword) *UserInfo
}
