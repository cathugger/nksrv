// +build !darwin,!dragonfly,!freebsd,!linux,!netbsd,!hurd,!solaris

package unixcryptcgo

func Crypt(key, salt string) string {
	panic("platform doesn't support proper crypt function")
}
