package cntp0

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
	MaxBits  = MaxBytes * 8
	MinBits  = 224
)

var (
	errUnknownAlgo = errors.New("unsupported digest option")
)

func parseIntParam(param string, def uint64) (n uint64, e error) {
	if param == "" {
		n = def
	} else {
		n, e = strconv.ParseUint(param, 10, 32)
		if e != nil {
			return
		}
		if n < MinBits || n > MaxBits || n%8 != 0 {
			e = errUnknownAlgo
			return
		}
	}
	return
}

// $CNTP0$SHA2-256$
// we're fed only "SHA2-256" in this case
// $CNTP0$BLAKE2b$256$
// we're fed "BLAKE2b" "256"
func PickDigest(algo, extra string) (d Digest, e error) {
	var n uint64
	switch algo {
	// BLAKE2
	case "blake2s":
		if extra == "" || extra == "256" {
			d = DigestBLAKE2s{}
		} else {
			e = errUnknownAlgo
		}
	case "blake2b":
		if extra == "" || extra == "512" {
			d = DigestBLAKE2b{s: 512}
		} else if extra == "384" {
			d = DigestBLAKE2b{s: 384}
		} else if extra == "256" {
			d = DigestBLAKE2b{s: 256}
		} else {
			e = errUnknownAlgo
		}
	// SHA2
	case "sha2-224":
		if extra == "" || extra == "224" {
			d = DigestSHA2_224{}
		} else {
			e = errUnknownAlgo
		}
	case "sha2-256":
		if extra == "" || extra == "256" {
			d = DigestSHA2_256{}
		} else {
			e = errUnknownAlgo
		}
	case "sha2-384":
		if extra == "" || extra == "384" {
			d = DigestSHA2_384{}
		} else {
			e = errUnknownAlgo
		}
	case "sha2-512":
		if extra == "" || extra == "512" {
			d = DigestSHA2_512{s: 512}
		} else if extra == "256" {
			d = DigestSHA2_512{s: 256}
		} else if extra == "224" {
			d = DigestSHA2_512{s: 224}
		} else {
			e = errUnknownAlgo
		}
	// SHA3/Keccak
	case "sha3-224":
		if extra == "" || extra == "224" {
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
