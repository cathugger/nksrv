package pibase

import "errors"

var (
	ErrNoSuchBoard  = errors.New("board does not exist")
	ErrNoSuchThread = errors.New("thread does not exist")
)
