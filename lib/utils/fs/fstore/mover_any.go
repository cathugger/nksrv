package fstore

import (
	"io"
	"os"
)

func NewMover(tmpstor *FStore) Mover {
	return Mover{tmpstor: tmpstor}
}

func (m *Mover) statCopyMove(from, to string) (err error) {

	// first check if exists
	_, err = os.Stat(to)
	if err == nil {
		// exists - don't overwrite
		return nil
	}
	if !os.IsNotExist(err) {
		// shouldn't happen
		return
	}

	// copy from
	rf, err := os.Open(from)
	if err != nil {
		return
	}
	defer rf.Close()

	// copy dest - tmp file for atomicity
	wf, err := m.tmpstor.NewFile("tmp", "mover-", "")
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			n := wf.Name()
			_ = wf.Close()
			_ = os.Remove(n)
		}
	}()

	// perform copy
	_, err = io.Copy(wf, rf)
	if err != nil {
		return
	}

	// sync to ensure consistency
	err = wf.Sync()
	if err != nil {
		return
	}

	fn := wf.Name()

	err = wf.Close()
	if err != nil {
		return
	}

	// perform rename
	err = os.Rename(fn, to)
	if err != nil {
		return
	}

	return
}
