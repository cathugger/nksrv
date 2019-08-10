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
	errFileTypeNotAllowed    = errors.New("file type not allowed")
	errTooLongTitle          = errors.New("too long title")
	errTooLongName           = errors.New("too long name")
	errDuplicateArticle      = errors.New("article with this ID already exists")
)

func errTooLongMessage(limit uint32) error {
	return fmt.Errorf("too long message (limit: %d)", limit)
}

func errTooMuchFiles(limit int32) error {
	return fmt.Errorf("too much files (limit: %d)", limit)
}

func errNotEnoughFiles(minimum int32) error {
	return fmt.Errorf("not enough files (minimum: %d)", minimum)
}

func errTooBigFileSingle(limit int64) error {
	return fmt.Errorf("one of files is too large (limit: %d)", limit)
}

func errTooBigFileAll(limit int64) error {
	return fmt.Errorf("files are too large (limit: %d)", limit)
}
