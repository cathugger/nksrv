package unixdescrypt

import (
	"testing"
	"math/rand"

	"nksrv/lib/unixdescrypt/unixcryptcgo"
)

func TestConst(t *testing.T) {
	type testcase struct {
		salt string
		key  string
		enc  string
	}
	var tests = [...]testcase{
		{"PQ", "test", "PQl1.p7BcJRuM"},
		{"xx", "much longer password here", "xxtHrOGVa3182"},
		{"xx", "much lon", "xxtHrOGVa3182"},
	}
	for i, tc := range tests {
		var salt [2]byte
		copy(salt[:], tc.salt)

		res := CryptDES([]byte(tc.key), salt, nil)

		if string(res) != tc.enc {
			t.Errorf("%d got %q expected %q\n", i, res, tc.enc)
		}
	}
}

func randsalt() (s [2]byte) {
	s[0] = ascii64[rand.Int31n(64)]
	s[1] = ascii64[rand.Int31n(64)]
	return
}

func randkey(b *[80]byte) int {
	i := int(rand.Int31n(80))
rerand:
	rand.Read(b[:i])
	for x := 0; x < i; x++ {
		if b[x] == 0 {
			goto rerand
		}
	}
	return i
}

func TestCompare(t *testing.T) {
	var k [80]byte
	var s [2]byte

	for i := 0; i < 20000; i++ {
		s = randsalt()
		ks := randkey(&k)

		res1 := unixcryptcgo.Crypt(string(k[:ks]), string(s[:]))
		res2 := CryptDES(k[:ks], s, nil)

		if res1 != string(res2) {
			t.Errorf(
				"%d expected %q got %q; k=%q s=%q\n",
				i, res1, res2, k[:ks], s[:])
		}
	}
}

func BenchmarkNative(b *testing.B) {
	var k [80]byte
	var s [2]byte
	var r [16]byte

	for n := 0; n < b.N; n++ {
		s = randsalt()
		ks := randkey(&k)
		_ = CryptDES(k[:ks], s, r[:0])
	}
}

func BenchmarkCGO(b *testing.B) {
	var k [80]byte
	var s [2]byte

	for n := 0; n < b.N; n++ {
		s = randsalt()
		ks := randkey(&k)
		_ = unixcryptcgo.Crypt(string(k[:ks]), string(s[:]))
	}
}
