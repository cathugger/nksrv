package psqlib

import (
	"errors"
	"fmt"
)

var (
	errNoSuchBoard           = errors.New("board does not exist")
	errNoSuchThread          = errors.New("thread does not exist")
	errNoSuchPage            = errors.New("page does not exist")
	errInvalidSubmission     = errors.New("invalid form submission")
	errBadSubmissionEncoding = errors.New("bad form submission encoding")
	errBadSubmissionChars    = errors.New("form submission contains control characters")
	errFileTypeNotAllowed    = errors.New("file type not allowed")
	errTooLongTitle          = errors.New("too long title")
	errTooLongName           = errors.New("too long name")
	errInvalidTripcode       = errors.New("invalid tripcode syntax (we expected 64 hex chars)")
	errDuplicateArticle      = errors.New("article with this ID already exists")
	errEmptyMsg              = errors.New("posting empty messages isn't allowed")
	errInvalidOptions        = errors.New("invalid options")
)

func errTooLongMessage(limit uint32) error {
	return fmt.Errorf("too long message (limit: %d)", limit)
}

func errTooMuchFiles(limit int32) error {
	if limit < 0 {
		limit = 0
	}
	return fmt.Errorf("too much files (limit: %d)", limit)
}

func errNotEnoughFiles(minimum int32) error {
	return fmt.Errorf("not enough files (minimum: %d)", minimum)
}

func errTooBigFileSingle(limit int64) error {
	return fmt.Errorf("one of files is too large (limit: %d bytes)", limit)
}

func errTooBigFileAll(limit int64) error {
	return fmt.Errorf("files are too large (limit: %d bytes)", limit)
}

// indicates that psql error is deadlock
type psqlDeadlockError struct {
	error
}

func (x psqlDeadlockError) Unwrap() error { return x.error }
