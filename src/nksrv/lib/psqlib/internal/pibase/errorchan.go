package pibase

func SendError(c chan<- error, e error) {
	// non-blocking send incase we have buffer space available
	select {
	case c <- e:
	default:
	}
}

func RecvError(c <-chan error) error {
	// non-blocking recv incase there's error buffered
	select {
	case e := <-c:
		return e
	default:
		return nil
	}
}
