// openbsd doesn't support crypt in des mode anymore
// +build darwin dragonfly freebsd linux hurd netbsd solaris

package unixcryptcgo

import (
	"sync"
	"unsafe"
)

// #cgo LDFLAGS: -lcrypt
// #define _XOPEN_SOURCE
// #define _DEFAULT_SOURCE
// #include <stdlib.h>
// #include <unistd.h>
import "C"

var cryptMu sync.Mutex

// Crypt invokes POSIX' crypt(3)
func Crypt(key, salt string) string {
	ckey := C.CString(key)
	csalt := C.CString(salt)
	cryptMu.Lock()
	out := C.GoString(C.crypt(ckey, csalt))
	cryptMu.Unlock()
	C.free(unsafe.Pointer(ckey))
	C.free(unsafe.Pointer(csalt))
	return out
}
