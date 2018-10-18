package demoib

import "errors"

var (
	errNoSuchBoard           = errors.New("board does not exist")
	errNoSuchThread          = errors.New("thread does not exist")
	errNoSuchPage            = errors.New("page does not exist")
	errInvalidSubmission     = errors.New("invalid form submission")
	errBadSubmissionEncoding = errors.New("bad form submission encoding")
	errTooMuchFiles          = errors.New("too much files")
	errTooBigFileSingle      = errors.New("one of files is too large")
	errTooBigFileAll         = errors.New("files are too large")
	errFileTypeNotAllowed    = errors.New("file type not allowed")
	errTooLongTitle          = errors.New("too long title")
	errTooLongMessage        = errors.New("too long message")
)
