// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package fstore

import "golang.org/x/sys/unix"

type Mover struct {
	nohardlink bool
	tmpstor    *FStore
}

func (m *Mover) HardlinkOrCopyIfNeededStable(from, to string) error {
	if !m.nohardlink {
		// hardlink syscall should abort incase file already exists
		// using unix.Link instead of os.Link is simpler
		e := unix.Link(from, to)
		if e == nil {
			return nil // OK
		}
		n := e.(unix.Errno)
		switch n {
		case unix.EEXIST:
			// already exists - OK
			return nil
		case unix.EXDEV, /* cross device */
			unix.EOPNOTSUPP, /* not supported by FS */
			unix.EPERM /* used by linux to mark no support */ :
			// will need to use copy
			m.nohardlink = true
		default:
			return e
		}
	}

	// fast path failed, do slow instead
	return m.statCopyMove(from, to)
}
