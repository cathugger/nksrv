package hashtools

import (
	"crypto/sha256"
	"hash"
	"io"
	"math/big"
	"sync"

	"github.com/zeebo/blake3"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/sys/cpu"
)

const HashLength = 28

type HashTypeIDType byte

const (
	// XXX in idea these could be for arbitrary lengths, but we have no practical need for that atm
	_ HashTypeIDType = iota // skip first to start with non-0

	SHA2_224    // can be faster if SHA2-256 crypto instructions are available
	BLAKE2b_224 // fastest on most 64bit CPUs without dedicated crypto instructions
	BLAKE3_224  // fastest on 32bit arm stuff (without SHA2 instructions) or AVX2 supporting stuff

	// SHA3_224 or SHAKE128_224 maybe once I get some hw to test w/ because w/o hw support they're so slow

	hashTypeIDMax = iota - 1
)

type hasherFactoryType struct {
	newHasher func() hash.Hash
}

var hasherFactories = [hashTypeIDMax]hasherFactoryType{
	{newHasher: sha256.New224},
	{newHasher: func() hash.Hash { x, _ := blake2b.New(28, nil); return x }},
	{newHasher: func() hash.Hash { return blake3.New() }},
}
var hashCtxPools [hashTypeIDMax]sync.Pool

var (
	defaultHashTypeID    HashTypeIDType
	defaultHasherFactory hasherFactoryType
	defaultHashCtxPool   *sync.Pool
)

type hashCtxType struct {
	h       hash.Hash
	copyBuf *[32 * 1024]byte
	x       big.Int
	strBuf  [44]byte // 28 hash bytes (224 bits) + 1 type byte = 29 bytes; floor(log36((2^224 - 1) + (2^224 * 3)) + 1 = 44; that remains true upto 10; 11 is 45 bytes
}

func getHashCtx(pool *sync.Pool, factory hasherFactoryType) *hashCtxType {
	s, _ := pool.Get().(*hashCtxType)
	if s != nil {
		s.h.Reset()
	} else {
		s = &hashCtxType{
			h:       factory.newHasher(),
			copyBuf: new([32 * 1024]byte),
		}
	}
	return s
}

func pickDefaultHash(typeID HashTypeIDType) {
	defaultHashTypeID = typeID
	defaultHasherFactory = hasherFactories[typeID-1]
	defaultHashCtxPool = &hashCtxPools[typeID-1]
}

func autoPickDefaultHash() {
	// currently only ARM64 because pretty much guaranteed gain
	// afaik x86_64 golang sha256 routine can't do SHA2 instructions (yet?)
	// XXX s390x?
	if cpu.ARM64.HasSHA2 {
		pickDefaultHash(SHA2_224)
		return
	}
	pickDefaultHash(BLAKE2b_224)
}

func init() { autoPickDefaultHash() }

// MakeFileHash returns textural representation of file hash for use in filename.
// It expects file to be seeked at 0.
func MakeFileHash(r io.Reader) (s string, e error) {

	hs := getHashCtx(defaultHashCtxPool, defaultHasherFactory)

	// NOTE: we're reusing strbuf (which is large enough) for hash destination. for 512-bit hashes this would be NOT large enough.
	// first byte - hash type
	hs.strBuf[0] = byte(defaultHashTypeID)

	// hash
	_, e = io.CopyBuffer(hs.h, r, hs.copyBuf[:])
	if e != nil {
		return
	}
	hs.h.Sum(hs.strBuf[1:][:0])

	// convert to base36 number and print it
	hs.x.SetBytes(hs.strBuf[:1+HashLength])
	xb := hs.x.Append(hs.strBuf[:0], 36) // this function apparently still allocates..

	// flip (we want front bits to be more variable)
	for i, j := 0, len(xb)-1; i < j; i, j = i+1, j-1 {
		xb[i], xb[j] = xb[j], xb[i]
	}

	// it's ready
	s = string(xb)

	defaultHashCtxPool.Put(hs)

	return
}

func MakeCustomFileHash(r io.Reader, typeID HashTypeIDType) (s string, h [HashLength]byte, e error) {

	factory := hasherFactories[typeID-1]
	pool := &hashCtxPools[typeID-1]

	hCtx := getHashCtx(pool, factory)

	// NOTE: we're reusing strbuf (which is large enough) for hash destination. for 512-bit hashes this would be NOT large enough.
	// first byte - hash type
	hCtx.strBuf[0] = byte(typeID)
	// hash
	_, e = io.CopyBuffer(hCtx.h, r, hCtx.copyBuf[:])
	if e != nil {
		return
	}
	hCtx.h.Sum(hCtx.strBuf[1:][:0])

	// save hash result
	copy(h[:], hCtx.strBuf[1:])

	// convert to base36 number and print it
	hCtx.x.SetBytes(hCtx.strBuf[:1+HashLength])
	xb := hCtx.x.Append(hCtx.strBuf[:0], 36) // this function apparently still allocates..

	// flip (we want front bits to be more variable)
	for i, j := 0, len(xb)-1; i < j; i, j = i+1, j-1 {
		xb[i], xb[j] = xb[j], xb[i]
	}

	// it's ready
	s = string(xb)

	pool.Put(hCtx)

	return
}
