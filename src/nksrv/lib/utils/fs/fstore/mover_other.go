// +build !aix,!darwin,!dragonfly,!freebsd,!linux,!netbsd,!openbsd,!solaris,!windows

type Mover struct {
	tmpstor *FStore
}

func (m *Mover) HardlinkOrCopyIfNeededStable(from, to string) error {
	return m.statCopyMove(from, to)
}
