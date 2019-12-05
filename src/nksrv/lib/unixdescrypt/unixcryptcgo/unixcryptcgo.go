package unixcryptcgo

import (
	"unsafe"
	"sync"
)

// #cgo LDFLAGS: -lcrypt
// #define _XOPEN_SOURCE
// #define _DEFAULT_SOURCE
// #include <stdlib.h>
// #include <unistd.h>
import "C"

var crypt_m sync.Mutex

func Crypt(key, salt string) string {
	ckey := C.CString(key)
	csalt := C.CString(salt)
	crypt_m.Lock()
	out := C.GoString(C.crypt(ckey, csalt))
	crypt_m.Unlock()
	C.free(unsafe.Pointer(ckey))
	C.free(unsafe.Pointer(csalt))
	return out
}
