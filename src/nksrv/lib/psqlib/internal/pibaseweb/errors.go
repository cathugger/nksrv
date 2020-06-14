package pibaseweb

import (
	"errors"
	"fmt"
)

var (
	ErrNoSuchBoard           = errors.New("board does not exist")
	ErrNoSuchThread          = errors.New("thread does not exist")
	ErrNoSuchPage            = errors.New("page does not exist")
	ErrInvalidSubmission     = errors.New("invalid form submission")
	ErrBadSubmissionEncoding = errors.New("bad form submission encoding")
	ErrBadSubmissionChars    = errors.New("form submission contains control characters")
	ErrFileTypeNotAllowed    = errors.New("file type not allowed")
	ErrTooLongTitle          = errors.New("too long title")
	ErrTooLongName           = errors.New("too long name")
	ErrInvalidTripcode       = errors.New("invalid tripcode syntax (we expected 64 hex chars)")
	//ErrDuplicateArticle      = errors.New("article with this ID already exists")
	ErrEmptyMsg       = errors.New("posting empty messages isn't allowed")
	ErrInvalidOptions = errors.New("invalid options")
)

func ErrTooLongMessage(limit uint32) error {
	return fmt.Errorf("too long message (limit: %d)", limit)
}

func ErrTooMuchFiles(limit int32) error {
	if limit < 0 {
		limit = 0
	}
	return fmt.Errorf("too much files (limit: %d)", limit)
}

func ErrNotEnoughFiles(minimum int32) error {
	return fmt.Errorf("not enough files (minimum: %d)", minimum)
}

func ErrTooBigFileSingle(limit int64) error {
	return fmt.Errorf("one of files is too large (limit: %d bytes)", limit)
}

func ErrTooBigFileAll(limit int64) error {
	return fmt.Errorf("files are too large (limit: %d bytes)", limit)
}

func ErrDuplicateFile(a, b int) error {
	return fmt.Errorf("duplicate file: %d is same as %d", a, b)
}
