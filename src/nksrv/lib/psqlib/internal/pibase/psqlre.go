package pibase

// indicates that psql error is deadlock
type PSQLRetriableError struct {
	error
}

func (x PSQLRetriableError) Unwrap() error { return x.error }
