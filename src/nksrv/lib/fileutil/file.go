package fileutil

import (
	"os"

	"golang.org/x/xerrors"
)

func SyncFileName(fname string) error {
	f, err := os.Open(fname)
	if err != nil {
		return xerrors.Errorf("failed os.Open: %v", fname, err)
	}
	err = f.Sync()
	c_err := f.Close()
	if err != nil {
		return xerrors.Errorf("failed f.Sync: %v", fname, err)
	}
	if c_err != nil {
		return xerrors.Errorf("failed f.Close: %v", fname, c_err)
	}
	return nil
}
