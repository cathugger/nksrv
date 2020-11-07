// +build !darwin,!dragonfly,!freebsd,!linux,!hurd,!netbsd,!solaris

package unixcryptcgo

func Crypt(key, salt string) string {
	panic("platform doesn't support proper crypt function")
}
