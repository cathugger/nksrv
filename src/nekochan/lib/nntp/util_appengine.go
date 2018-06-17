// +build appengine appenginevm

package nntp

func unsafeStrToBytes(s string) []byte {
	return []byte(s)
}

func unsafeBytesToStr(b []byte) string {
	return string(b)
}
