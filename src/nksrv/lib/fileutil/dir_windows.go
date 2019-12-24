// +build windows

package fileutil

// windows can't do sync on dirs
func syncDir(dir string) error { return nil }
