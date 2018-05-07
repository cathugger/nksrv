// +build appengine appenginevm

package webib0

func unsafeStrToBytes(s string) []byte {
	return []byte(s)
}

func unsafeBytesToStr(b []byte) string {
	return string(b)
}
