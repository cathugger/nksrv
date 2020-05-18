package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"hash/crc32"
	"hash/crc64"
	"io"
	"testing"

	"github.com/minio/highwayhash"
	"golang.org/x/crypto/blake2b"
	"lukechampine.com/blake3"
)

var crc32c = crc32.MakeTable(crc32.Castagnoli)
var crc64e = crc64.MakeTable(crc64.ECMA)

const sizeSmall = 420
const sizeBig = 2 * 1024 * 1024

var smallBuf, bigBuf []byte

var hhkey [32]byte

func init() {
	smallBuf = make([]byte, sizeSmall)
	bigBuf = make([]byte, sizeBig)
	rand.Read(smallBuf)
	rand.Read(bigBuf)
}

func BenchmarkCRC32_small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := crc32.New(crc32c)
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum32()
	}
}

func BenchmarkCRC32_big(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := crc32.New(crc32c)
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum32()
	}
}

func BenchmarkCRC64_small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := crc64.New(crc64e)
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum64()
	}
}

func BenchmarkCRC64_big(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := crc64.New(crc64e)
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum64()
	}
}

func BenchmarkHH64_small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h, _ := highwayhash.New64(hhkey[:])
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum64()
	}
}

func BenchmarkHH64_big(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h, _ := highwayhash.New64(hhkey[:])
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum64()
	}
}

func BenchmarkHH128_small(b *testing.B) {
	var s [16]byte
	for i := 0; i < b.N; i++ {
		h, _ := highwayhash.New128(hhkey[:])
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkHH128_big(b *testing.B) {
	var s [16]byte
	for i := 0; i < b.N; i++ {
		h, _ := highwayhash.New128(hhkey[:])
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkMD5_small(b *testing.B) {
	var s [16]byte
	for i := 0; i < b.N; i++ {
		h := md5.New()
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkMD5_big(b *testing.B) {
	var s [16]byte
	for i := 0; i < b.N; i++ {
		h := md5.New()
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkSHA1_small(b *testing.B) {
	var s [20]byte
	for i := 0; i < b.N; i++ {
		h := sha1.New()
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkSHA1_big(b *testing.B) {
	var s [20]byte
	for i := 0; i < b.N; i++ {
		h := sha1.New()
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkSHA2_224_small(b *testing.B) {
	var s [28]byte
	for i := 0; i < b.N; i++ {
		h := sha256.New224()
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkSHA2_224_big(b *testing.B) {
	var s [28]byte
	for i := 0; i < b.N; i++ {
		h := sha256.New224()
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkBLAKE2b_224_small(b *testing.B) {
	var s [28]byte
	for i := 0; i < b.N; i++ {
		h, _ := blake2b.New(28, nil)
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkBLAKE2b_224_big(b *testing.B) {
	var s [28]byte
	for i := 0; i < b.N; i++ {
		h, _ := blake2b.New(28, nil)
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkBLAKE3_small(b *testing.B) {
	var s [32]byte
	for i := 0; i < b.N; i++ {
		h := blake3.New(32, nil)
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum(s[:0])
	}
}

func BenchmarkBLAKE3_big(b *testing.B) {
	var s [32]byte
	for i := 0; i < b.N; i++ {
		h := blake3.New(32, nil)
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum(s[:0])
	}
}
