package psqlib

import (
	"fmt"
	"os"
	"runtime/debug"
)

func sqlError(err error, when string) error {
	fmt.Fprintf(os.Stderr, "[sql] error on %s: %v\n", when, err)
	debug.PrintStack()
	return fmt.Errorf("error on %s: %v", when, err)
}
