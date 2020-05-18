package main

import (
	"bytes"
	"crypto/rand"
	"hash/crc32"
	"hash/crc64"
	"io"
	"testing"

	"github.com/minio/highwayhash"
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

func BenchmarkHH_small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h, _ := highwayhash.New64(hhkey[:])
		io.Copy(h, bytes.NewReader(smallBuf))
		_ = h.Sum64()
	}
}

func BenchmarkHH_big(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h, _ := highwayhash.New64(hhkey[:])
		io.Copy(h, bytes.NewReader(bigBuf))
		_ = h.Sum64()
	}
}
