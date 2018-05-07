// +build appengine appenginevm

package com0

func unsafeStrToBytes(s string) []byte {
	return []byte(s)
}

func unsafeBytesToStr(b []byte) string {
	return string(b)
}
