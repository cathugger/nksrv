package xos

import "os"

func IsClosed(err error) bool {
	if err == os.ErrClosed {
		return true
	}
	err = underlyingError(err)
	if err == nil {
		return false
	}
	return contains(err.Error(), "closed")
}

// underlyingError returns the underlying error for known os error types.
// for unknown types it returns nil.
func underlyingError(err error) error {
	switch err := err.(type) {
	case *os.PathError:
		return err.Err
	case *os.LinkError:
		return err.Err
	case *os.SyscallError:
		return err.Err
	}
	return nil
}

// contains is a local version of strings.Contains.
// It knows len(sep) > 1.
func contains(s, sep string) bool {
	n := len(sep)
	c := sep[0]
	for i := 0; i+n <= len(s); i++ {
		if s[i] == c && s[i:i+n] == sep {
			return true
		}
	}
	return false
}
