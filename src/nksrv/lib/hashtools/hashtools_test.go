package hashtools

import (
	"io"
	"testing"
)

type zeroreader struct {
	n int
}

var zbuf [65536]byte

func (r *zeroreader) Read(b []byte) (n int, e error) {
	if len(b) > r.n {
		b = b[:r.n]
	}
	n = copy(b, zbuf[:])
	r.n -= n
	if r.n == 0 {
		e = io.EOF
	}
	return
}

func dotest(b *testing.B, exp string) {
	for i := 0; i < b.N; i++ {
		s, e := MakeFileHash(&zeroreader{2 << 20})
		if e != nil {
			panic("x")
		}
		if exp != "" && exp != s {
			b.Errorf("exp %q != s %q", exp, s)
		}
	}
}

var hexp = [...]string{
	"73xc5t80vzz32ubrgs2198rm8jbrs5r5u9sov02qlea3",
	"x",
	"4ffm46fu1rbaio6i4yo4uvjn2e6r8dmi2r59wzsonx7b",
}

func doxtest(b *testing.B, id byte) {
	b.StopTimer()
	pickhash(id)
	b.StartTimer()

	dotest(b, hexp[id-1])
}

func BenchmarkHashAuto(b *testing.B) {
	b.StopTimer()
	autopickhash()
	b.StartTimer()

	dotest(b, "")
}

func BenchmarkHashBLAKE2b_224(b *testing.B) {
	doxtest(b, ht_BLAKE2b_224)
}

func BenchmarkHashSHA2_224(b *testing.B) {
	doxtest(b, ht_SHA2_224)
}
