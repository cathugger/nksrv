package nntp

type ClientPassword string

type UserPassProvider interface {
	// given username, returns info and password
	// can return empty pass which would mean there's no pass
	NNTPUserPassByName(name string) (*UserInfo, string)
	// given challenge password and client password, returns whether they match
	NNTPCheckPass(ch string, rpass ClientPassword) bool
	// given username and correct password, returns info
	NNTPCheckUserPass(username string, rpass ClientPassword) *UserInfo
}
