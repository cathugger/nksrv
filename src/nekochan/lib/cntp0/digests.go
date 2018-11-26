package cntp0

import (
	"crypto/sha256"
	"crypto/sha512"
	"io"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/sha3"
)

// digesters

type Digest interface {
	Hasher() Hasher
	WriteIdentifier(w io.Writer)
}

// BLAKE2s
type DigestBLAKE2s struct{}

func (DigestBLAKE2s) Hasher() (h Hasher) {
	h.h, _ = blake2s.New256(nil)
	return
}

var nameBLAKE2s = []byte("BLAKE2s")

func (DigestBLAKE2s) WriteIdentifier(w io.Writer) {
	w.Write(nameBLAKE2s)
}

// BLAKE2b
type DigestBLAKE2b struct {
	s uint32 // in bytes
}

func (d DigestBLAKE2b) Hasher() (h Hasher) {
	switch d.s * 8 {
	case 512:
		h.h, _ = blake2b.New512(nil)
	case 384:
		h.h, _ = blake2b.New384(nil)
	case 256:
		h.h, _ = blake2b.New256(nil)
	}
	return
}

var nameBLAKE2b = []byte("BLAKE2b")

func (d DigestBLAKE2b) WriteIdentifier(w io.Writer) {
	w.Write(nameBLAKE2b)
}

// SHA2
// 224
type DigestSHA2_224 struct{}

func (DigestSHA2_224) Hasher() Hasher {
	return Hasher{h: sha256.New224()}
}

var nameSHA2_224 = []byte("SHA2-224")

func (DigestSHA2_224) WriteIdentifier(w io.Writer) {
	w.Write(nameSHA2_224)
}

// 256
type DigestSHA2_256 struct{}

func (DigestSHA2_256) Hasher() Hasher {
	return Hasher{h: sha256.New()}
}

var nameSHA2_256 = []byte("SHA2-256")

func (DigestSHA2_256) WriteIdentifier(w io.Writer) {
	w.Write(nameSHA2_256)
}

// 384
type DigestSHA2_384 struct{}

func (DigestSHA2_384) Hasher() Hasher {
	return Hasher{h: sha512.New384()}
}

var nameSHA2_384 = []byte("SHA2-384")

func (DigestSHA2_384) WriteIdentifier(w io.Writer) {
	w.Write(nameSHA2_384)
}

// 512
type DigestSHA2_512 struct {
	s uint32 // in bytes
}

func (d DigestSHA2_512) Hasher() (h Hasher) {
	switch d.s * 8 {
	case 512:
		h.h = sha512.New()
	case 256:
		h.h = sha512.New512_256()
	case 224:
		h.h = sha512.New512_224()
	}
	return
}

var nameSHA2_512 = []byte("SHA2-512")

func (d DigestSHA2_512) WriteIdentifier(w io.Writer) {
	w.Write(nameSHA2_512)
}

// SHA3
// 224
type DigestSHA3_224 struct{}

func (DigestSHA3_224) Hasher() Hasher {
	return Hasher{h: sha3wrap{sha3.New224().(keccakstate)}}
}

var nameSHA3_224 = []byte("SHA3-224")

func (DigestSHA3_224) WriteIdentifier(w io.Writer) {
	w.Write(nameSHA3_224)
}

// 256
type DigestSHA3_256 struct{}

func (DigestSHA3_256) Hasher() Hasher {
	return Hasher{h: sha3wrap{sha3.New256().(keccakstate)}}
}

var nameSHA3_256 = []byte("SHA3-256")

func (DigestSHA3_256) WriteIdentifier(w io.Writer) {
	w.Write(nameSHA3_256)
}

// 384
type DigestSHA3_384 struct{}

func (DigestSHA3_384) Hasher() Hasher {
	return Hasher{h: sha3wrap{sha3.New384().(keccakstate)}}
}

var nameSHA3_384 = []byte("SHA3-384")

func (DigestSHA3_384) WriteIdentifier(w io.Writer) {
	w.Write(nameSHA3_384)
}

// 512
type DigestSHA3_512 struct{}

func (DigestSHA3_512) Hasher() Hasher {
	return Hasher{h: sha3wrap{sha3.New512().(keccakstate)}}
}

var nameSHA3_512 = []byte("SHA3-512")

func (DigestSHA3_512) WriteIdentifier(w io.Writer) {
	w.Write(nameSHA3_512)
}

// SHAKE
type DigestSHAKE_128 struct {
	s uint32 // in bytes
}

func (d DigestSHAKE_128) Hasher() Hasher {
	return Hasher{
		h: shakewrap{
			keccakstate: sha3.NewShake128().(keccakstate),
			size:        d.s,
		},
	}
}

var nameSHAKE_128 = []byte("SHAKE-128")

func (d DigestSHAKE_128) WriteIdentifier(w io.Writer) {
	w.Write(nameSHAKE_128)
}

type DigestSHAKE_256 struct {
	s uint32 // in bytes
}

func (d DigestSHAKE_256) Hasher() Hasher {
	return Hasher{
		h: shakewrap{
			keccakstate: sha3.NewShake256().(keccakstate),
			size:        d.s,
		},
	}
}

var nameSHAKE_256 = []byte("SHAKE-256")

func (d DigestSHAKE_256) WriteIdentifier(w io.Writer) {
	w.Write(nameSHAKE_256)
}
