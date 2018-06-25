// +build appengine appenginevm

package testsrv

func unsafeStrToBytes(s string) []byte {
	return []byte(s)
}

func unsafeBytesToStr(b []byte) string {
	return string(b)
}
