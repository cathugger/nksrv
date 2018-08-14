//

package fileutil

import (
	"os"
)

func renameNoClobber(oldpath, newpath string) error {
	// if already exists, bail out
	// I'm aware this isn't race-free, but should be good enough for most cases and hopefuly overwrite isn't fatal
	// TODO: use OS-dependent syscalls if possible
	_, err := os.Stat(newpath)
	if err == nil || !os.IsNotExist(err) {
		if err == nil {
			err = os.ErrExist
		}
		return err
	}

	// perform rename
	return os.Rename(oldpath, newpath)
}
