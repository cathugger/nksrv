package hashtools

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"hash"
	"io"
	"math/big"
	"sync"

	//"github.com/zeebo/blake3"
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
	newHasher func() hash.Hash
}

const (
	// XXX in idea these could be for arbitrary lengths, but we have no practical need for that atm
	_              = iota // skip first to start with non-0
	ht_SHA2_224           // can be faster if SHA2-256 crypto instructions are available
	ht_BLAKE2b_224        // fastest on most 64bit CPUs without dedicated crypto instructions
	//_                     // ht_BLAKE2s_224 // faster on weak 32bit CPUs -- waiting on when x/crypto provides New224 or equivalent
	//_                     // ht_SHA3_224 // maybe once I get some hardware to test
	//ht_BLAKE3_224
	ht_max = iota - 1
)

var h_hashes = [ht_max]fhash{
	{newHasher: func() hash.Hash { return sha256.New224() }},
	{newHasher: func() hash.Hash { x, _ := blake2b.New(28, nil); return x }},
	//{newHasher: func() hash.Hash { return blake3.New() }},
}
var h_pools [ht_max]sync.Pool
var h_use_id byte
var h_use_hash fhash
var h_use_pool *sync.Pool

type hashstuff struct {
	h       hash.Hash
	copybuf *[32 * 1024]byte
	x       big.Int
	strbuf  [44]byte // 28 type bytes (224 bits) + 1 type byte = 29 bytes; floor(log36((2^224 - 1) + (2^224 * 3)) + 1 = 44; that remains true upto 10; 11 is 45 bytes

}

func gethasher() *hashstuff {
	s, _ := h_use_pool.Get().(*hashstuff)
	if s != nil {
		s.h.Reset()
	} else {
		s = &hashstuff{
			h:       h_use_hash.newHasher(),
			copybuf: new([32 * 1024]byte),
		}
	}
	return s
}

func pickhash(id byte) {
	h_use_id = id
	h_use_hash = h_hashes[id-1]
	h_use_pool = &h_pools[id-1]
}

func autopickhash() {
	// currently only ARM64 because pretty much guaranteed gain
	// afaik x86_64 golang sha256 routine can't do SHA2 instructions (yet?)
	// XXX s390x?
	// XXX SHA3 for ones where it's possible?
	if cpu.ARM64.HasSHA2 {
		pickhash(ht_SHA2_224)
		return
	}
	pickhash(ht_BLAKE2b_224)
}

func init() { autopickhash() }

// MakeFileHash returns textural representation of file hash for use in filename.
// It expects file to be seeked at 0.
func MakeFileHash(r io.Reader) (s string, e error) {
	hs := gethasher()
	// first byte - hash type
	hs.strbuf[0] = h_use_id
	// hash
	_, e = io.CopyBuffer(hs.h, r, hs.copybuf[:])
	if e != nil {
		return
	}
	hs.h.Sum(hs.strbuf[1:][:0])

	// convert to base36 number and print it
	hs.x.SetBytes(hs.strbuf[:1+hashLen])
	xb := hs.x.Append(hs.strbuf[:0], 36) // this function apparently still allocates..

	// flip (we want front bits to be more variable)
	for i, j := 0, len(xb)-1; i < j; i, j = i+1, j-1 {
		xb[i], xb[j] = xb[j], xb[i]
	}

	// it's ready
	s = string(xb)

	h_use_pool.Put(hs)

	return
}
