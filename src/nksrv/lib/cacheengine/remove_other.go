// +build !linux,!freebsd,!netbsd,!openbsd,!dragonfly,!solaris,!darwin

package cacheengine

// plan9 axes existing file handles on remove
// windows refuses to remove
const safeToRemoveOpen = false
