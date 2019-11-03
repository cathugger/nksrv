package psqlib

import (
	"reflect"
	"unsafe"
)

func unsafeStrToBytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

func unsafeBytesToStr(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// fuck golang

func maxInt(a, b int) int {
	if a >= b {
		return a
	} else {
		return b
	}
}

func minInt(a, b int) int {
	if a <= b {
		return a
	} else {
		return b
	}
}
