package cntp0

// ID 0 should be used only for testing

import (
	"errors"
	"hash"
	"strconv"
)

type Hasher struct {
	h hash.Hash
}

const (
	MaxBytes = 128
	MinBits  = 224
	MaxBits  = MaxBytes * 8
)

var (
	errUnknownAlgo = errors.New("unsupported digest option")
)

func parseBlake2s(size

var hashes := map[string]struct{
	defaultSize int
	parseFunc func(Digest, error)
}{
	"blake2s":{
		defaultSize: 256/8,
		parseFunc: func(sz int) (d Digest, e error) {
			if size == 256/8 {
				d = DigestBLAKE2s{}
			} else {
				e = errUnknownAlgo
			}
		},
	},
	"blake2b":{
		defaultSize: 512/8,
		
	},
}

// $CNTP0$SHA2-256$
// we're fed only "SHA2-256" in this case
// we also are fed size in bytes
func PickDigest(algo string, size int) (d Digest, e error) {
	var n uint64
	switch algo {
	// BLAKE2
	case "blake2b":
		if size == 0 {
			size = 512/8
		}
		if size == 512/8 || size == 384/8 || size == 256/8 {
			d = DigestBLAKE2b{s: size}
		} else {
			e = errUnknownAlgo
		}
	// SHA2
	case "sha2-224":
		if size == 0 {
			size = 224/8
		}
		if size == 224/8 {
			d = DigestSHA2_224{}
		} else {
			e = errUnknownAlgo
		}
	case "sha2-256":
		if size == 0 {
			size = 256/8
		}
		if size == 256/8 {
			d = DigestSHA2_256{}
		} else {
			e = errUnknownAlgo
		}
	case "sha2-384":
		if size == 0 {
			size = 384/8
		}
		if size == 384/8 {
			d = DigestSHA2_384{}
		} else {
			e = errUnknownAlgo
		}
	case "sha2-512":
		if size == 0 {
			size = 512/8
		}
		if size == 512/8 || size==256/8 || size==224/8
			d = DigestSHA2_512{s: size}
		} else {
			e = errUnknownAlgo
		}
	// SHA3/Keccak
	case "sha3-224":
		if size == 0 {
			size = 224/8
		}
		if size == 224/8 {
			d = DigestSHA3_224{}
		} else {
			e = errUnknownAlgo
		}
	case "sha3-256":
		if extra == "" || extra == "256" {
			d = DigestSHA3_256{}
		} else {
			e = errUnknownAlgo
		}
	case "sha3-384":
		if extra == "" || extra == "384" {
			d = DigestSHA3_384{}
		} else {
			e = errUnknownAlgo
		}
	case "sha3-512":
		if extra == "" || extra == "512" {
			d = DigestSHA3_512{}
		} else {
			e = errUnknownAlgo
		}
	case "shake-128":
		n, e = parseIntParam(extra, 256)
		if e == nil {
			d = DigestSHAKE_128{s: uint32(n)}
		}
	case "shake-256":
		n, e = parseIntParam(extra, 512)
		if e == nil {
			d = DigestSHAKE_256{s: uint32(n)}
		}
	default:
		e = errUnknownAlgo
	}
	return
}
