package hashtools

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"hash"
	"io"
	"math/big"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/sys/cpu"
)

// like normal base32 just lowercase and without padding
var LowerBase32Set = "abcdefghijklmnopqrstuvwxyz234567"
var LowerBase32Enc = base32.
	NewEncoding(LowerBase32Set).
	WithPadding(base32.NoPadding)

// lowecase base32 set which preserves sorting order without padding
var LowerBase32HexSet = "0123456789abcdefghijklmnopqrstuv"
var LowerBase32HexEnc = base32.
	NewEncoding(LowerBase32HexSet).
	WithPadding(base32.NoPadding)

// custom base64 set (preserves sort order) without padding
var SBase64Set = "-" +
	"0123456789" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"_" +
	"abcdefghijklmnopqrstuvwxyz"
var SBase64Enc = base64.
	NewEncoding(SBase64Set).
	WithPadding(base64.NoPadding)

const hashLen = 28

type fhash struct {
	newHasher func() (hash.Hash, error)
}

const (
	// XXX in idea these could be for arbitrary lengths, but we have no practical need for that atm
	_              = iota // skip first to start with non-0
	ht_BLAKE2b_224        // fastest on most 64bit CPUs without dedicated crypto instructions
	ht_SHA2_224           // can be faster if SHA2-256 crypto instructions are available
	// XXX SHA3/SHAKE? maybe when there is hw to test...
)

var h_selecta = [...]fhash{
	{newHasher: func() (hash.Hash, error) { return blake2b.New(28, nil) }},
	{newHasher: func() (hash.Hash, error) { return sha256.New224(), nil }},
}
var h_use_id byte
var h_use fhash

func pickhash(id byte) {
	h_use_id = id
	h_use = h_selecta[id-1]
}
func init() {
	// currently only ARM64 because pretty much guaranteed gain
	// afaik x86_64 sha256 routine can't do SHA2 instructions
	// XXX s390x?
	if cpu.ARM64.HasSHA2 {
		pickhash(ht_SHA2_224)
		return
	}
	pickhash(ht_BLAKE2b_224)
}

// MakeFileHash returns textural representation of file hash for use in filename.
// It expects file to be seeked at 0.
func MakeFileHash(r io.Reader) (s string, e error) {
	const slen = 48 // technically 44 for 224bit+mark, but wont hurt to have a bit more
	var b [slen]byte

	b[0] = h_use_id
	// hash
	h, e := h_use.newHasher()
	if e != nil {
		panic("newHasher(): " + e.Error())
	}
	_, e = io.Copy(h, r)
	if e != nil {
		return
	}
	h.Sum(b[1:][:0])

	// convert to base36 number and print it
	var x big.Int
	x.SetBytes(b[:1+hashLen])
	xb := x.Append(b[:0], 36)

	// flip (we want front bits to be more variable)
	for i, j := 0, len(xb)-1; i < j; i, j = i+1, j-1 {
		xb[i], xb[j] = xb[j], xb[i]
	}

	// it's ready
	s = string(xb)

	return
}
