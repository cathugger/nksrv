// +build !windows

package fileutil

import (
	"fmt"
	"os"
)

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("failed os.Open dir: %v", err)
	}
	err = f.Sync()
	c_err := f.Close()
	if err != nil {
		return fmt.Errorf("failed f.Sync dir: %v", err)
	}
	if c_err != nil {
		return fmt.Errorf("failed f.Close dir: %v", c_err)
	}
	return nil
}
