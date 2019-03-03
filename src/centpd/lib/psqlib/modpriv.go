package psqlib

type ModPriv int

const (
	ModPrivNone ModPriv = iota
	ModPrivMod
	_ModPrivMax
)

var ModPrivS = [_ModPrivMax]string{
	ModPrivNone: "none",
	ModPrivMod:  "mod",
}

var ModPrivM map[string]ModPriv

func init() {
	ModPrivM = make(map[string]ModPriv)
	for i, v := range ModPrivS {
		ModPrivM[v] = ModPriv(i)
	}
}

func StringToModPriv(s string) ModPriv {
	return ModPrivM[s]
}

func (t ModPriv) String() string {
	return ModPrivS[t]
}
