package psqlib

import (
	"runtime/debug"
	"fmt"
	"os"
)

func sqlError(err error, when string) error {
	fmt.Fprintf(os.Stderr, "[sql] error on %s: %v\n", when, err)
	debug.PrintStack()
	return fmt.Errorf("error on %s: %v", when, err)
}
