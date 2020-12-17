package hashtools

import (
	"io"
	"testing"
)

type zeroreader struct {
	n int64
}

var zbuf [65536]byte

func (r *zeroreader) Read(b []byte) (n int, e error) {
	if int64(len(b)) > r.n {
		b = b[:r.n]
	}
	n = copy(b, zbuf[:])
	r.n -= int64(n)
	if r.n == 0 {
		e = io.EOF
	}
	return
}

func dotest(t *testing.T, exp string, sizeidx int) {
	got, e := MakeFileHash(&zeroreader{sizes[sizeidx]})
	if e != nil {
		t.Logf("MakeFileHash err: %v", e)
		t.FailNow()
	}
	if exp != "" && exp != got {
		t.Errorf("exp %q != got %q", exp, got)
	}
}

func dobench(b *testing.B, exp string, sizeidx int) {
	for i := 0; i < b.N; i++ {
		got, e := MakeFileHash(&zeroreader{sizes[sizeidx]})
		if e != nil {
			b.Logf("MakeFileHash err: %v", e)
			b.FailNow()
		}
		if exp != "" && exp != got {
			b.Errorf("exp %q != got %q", exp, got)
		}
	}
}

const (
	sizebigg = iota
	sizesmol
)

var hexp = [...][ht_max]string{
	{
		"ms0bmq9elpsvml0zh5klbwjcl5fbevm3uxigmyeabcs5", // SHA2-224
		"banl818ny8i178t8z7x93fwo51c5zumjmr8mb5v6bcm6", // BLAKE2b-224
		"9ye8a31d0pffo0hsn6dp48psp5sk6jteiqhzagjt3hr9", // BLAKE3
	},
	{
		"scdpymqbxyn8zwgrevmz7227jhzipuwa6d94dsp10pf4", // SHA2-224
		"1qvyte14cv8a0qyu4s8k2miv0600ggd2okv9r11gbfw8", // BLAKE2b-224
		"9f34195rc24ar3xzdsy00wn42fani1c1zupo6jk7a1x9", // BLAKE3
	},
}

func doxtest(t *testing.T, id byte, sizeidx int) {
	pickhash(id)
	dotest(t, hexp[sizeidx][id-1], sizeidx)
}

func doxbench(b *testing.B, id byte, sizeidx int) {
	b.StopTimer()
	pickhash(id)
	b.StartTimer()

	dobench(b, hexp[sizeidx][id-1], sizeidx)
}

var sizes = [...]int64{4 << 20, 16 << 10}

func TestHashAutoSmol(t *testing.T) {
	autopickhash()
	dotest(t, "", sizesmol)
}

func BenchmarkHashAutoSmol(b *testing.B) {
	b.StopTimer()
	autopickhash()
	b.StartTimer()

	dobench(b, "", sizesmol)
}
func BenchmarkHashAutoBigg(b *testing.B) {
	b.StopTimer()
	autopickhash()
	b.StartTimer()

	dobench(b, "", sizebigg)
}

func TestHashSHA2_224(t *testing.T) {
	doxtest(t, ht_SHA2_224, sizesmol)
}
func BenchmarkHashSHA2_224_Smol(b *testing.B) {
	doxbench(b, ht_SHA2_224, sizesmol)
}
func BenchmarkHashSHA2_224_Bigg(b *testing.B) {
	doxbench(b, ht_SHA2_224, sizebigg)
}

func TestHashBLAKE2b_224(t *testing.T) {
	doxtest(t, ht_BLAKE2b_224, sizesmol)
}
func BenchmarkHashBLAKE2b_224_Smol(b *testing.B) {
	doxbench(b, ht_BLAKE2b_224, sizesmol)
}
func BenchmarkHashBLAKE2b_224_Bigg(b *testing.B) {
	doxbench(b, ht_BLAKE2b_224, sizebigg)
}

func TestHashBLAKE3_224(t *testing.T) {
	doxtest(t, ht_BLAKE3_224, sizesmol)
}
func BenchmarkHashBLAKE3_224_Smol(b *testing.B) {
	doxbench(b, ht_BLAKE3_224, sizesmol)
}
func BenchmarkHashBLAKE3_224_Bigg(b *testing.B) {
	doxbench(b, ht_BLAKE3_224, sizebigg)
}
