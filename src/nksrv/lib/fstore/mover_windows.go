// +build windows

package fstore

import (
	"os"

	"golang.org/x/sys/windows"
)

type Mover struct {
	nohardlink bool
	tmpstor    *FStore
}

// untested
const hardlinkwin = false

func (m *Mover) HardlinkOrCopyIfNeededStable(from, to string) error {
	if hardlinkwin && !m.nohardlink {
		// hardlink syscall should abort incase file already exists
		e := os.Link(from, to)
		if e == nil {
			return nil // OK
		}
		le := e.(*os.LinkError)
		n := le.Err.(windows.Errno)
		// TODO: determine correct code(s) by testing -- can't find proper spec
		switch n {
		case windows.ERROR_FILE_EXISTS:
			// already exists - OK
			return nil
		case windows.ERROR_NOT_SUPPORTED:
			// will need to use copy
			m.nohardlink = true
		default:
			return e
		}
	}

	// fast path failed, do slow instead
	return m.statCopyMove(from, to)
}
