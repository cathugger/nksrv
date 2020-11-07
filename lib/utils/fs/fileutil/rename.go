package fileutil

// attempts to rename without overwriting existing file
func RenameNoClobber(oldpath, newpath string) error {
	return renameNoClobber(oldpath, newpath)
}
