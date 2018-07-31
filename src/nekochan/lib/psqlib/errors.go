package psqlib

import "errors"

var (
	errNoSuchBoard       = errors.New("board does not exist")
	errNoSuchThread      = errors.New("thread does not exist")
	errNoSuchPage        = errors.New("page does not exist")
	errInvalidSubmission = errors.New("invalid form submission")
)
