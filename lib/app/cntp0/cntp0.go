package cntp0

// ID 0 should be used only for testing

import (
	"errors"
	"hash"
	"strings"
)

type Hasher struct {
	h hash.Hash
}

const (
	MaxBits  = MaxBytes * 8
	MaxBytes = 128
	MinBits  = 224
	MinBytes = MinBits / 8
)

var (
	errUnknownAlgo = errors.New("unsupported digest option")
)

type hasher struct {
	defaultSize int
	parseFunc   func(sz int) (Digest, error)
}

func checkDefault(sz, def int, d Digest) (Digest, error) {
	if sz == def {
		return d, nil
	} else {
		return nil, errUnknownAlgo
	}
}

var hashes = map[string]hasher{
	// BLAKE2s
	"blake2s": {
		defaultSize: 256 / 8,
		parseFunc: func(sz int) (Digest, error) {
			return checkDefault(sz, 256/8, DigestBLAKE2s{})
		},
	},

	// BLAKE2b
	"blake2b": {
		defaultSize: 512 / 8,
		parseFunc: func(size int) (d Digest, e error) {
			if sz >= MinBytes && sz <= 512/8 && sz%8 == 0 {
				d = DigestBLAKE2b{s: size}
			} else {
				e = errUnknownAlgo
			}
			return
		},
	},

	// SHA2
	"sha2-224": {
		defaultSize: 224 / 8,
		parseFunc: func(sz int) (Digest, error) {
			return checkDefault(sz, 224/8, DigestSHA2_224{})
		},
	},
	"sha2-256": {
		defaultSize: 256 / 8,
		parseFunc: func(sz int) (Digest, error) {
			return checkDefault(sz, 256/8, DigestSHA2_256{})
		},
	},
	"sha2-384": {
		defaultSize: 384 / 8,
		parseFunc: func(sz int) (Digest, error) {
			return checkDefault(sz, 384/8, DigestSHA2_384{})
		},
	},
	"sha2-512": {
		defaultSize: 512 / 8,
		parseFunc: func(sz int) (Digest, error) {
			if sz == 512/8 || sz == 384/8 || sz == 256/8 {
				d = DigestSHA2_512{s: sz}
			} else {
				e = errUnknownAlgo
			}
			return
		},
	},

	// SHA3
	"sha3-224": {
		defaultSize: 224 / 8,
		parseFunc: func(sz int) (Digest, error) {
			return checkDefault(sz, 224/8, DigestSHA3_224{})
		},
	},
	"sha3-256": {
		defaultSize: 256 / 8,
		parseFunc: func(sz int) (Digest, error) {
			return checkDefault(sz, 256/8, DigestSHA3_256{})
		},
	},
	"sha3-384": {
		defaultSize: 384 / 8,
		parseFunc: func(sz int) (Digest, error) {
			return checkDefault(sz, 384/8, DigestSHA3_384{})
		},
	},
	"sha3-512": {
		defaultSize: 512 / 8,
		parseFunc: func(sz int) (Digest, error) {
			return checkDefault(sz, 512/8, DigestSHA3_512{})
		},
	},
	"shake-128": {
		defaultSize: 256 / 8,
		parseFunc: func(sz int) (Digest, error) {
			if sz >= MinBytes && sz <= MaxBytes && sz%8 == 0 {
				d = DigestSHAKE_128{s: sz}
			} else {
				e = errUnknownAlgo
			}
			return
		},
	},
	"shake-256": {
		defaultSize: 512 / 8,
		parseFunc: func(sz int) (Digest, error) {
			if sz >= MinBytes && sz <= MaxBytes && sz%8 == 0 {
				d = DigestSHAKE_256{s: sz}
			} else {
				e = errUnknownAlgo
			}
			return
		},
	},
}

// $CNTP0$SHA2-256$
// we're fed only "SHA2-256" in this case
func PickDigest(algo string, size int) (d Digest, e error) {
	lalgs := strings.ToLower(algo)
	alg, ok := hashes[lalgs]
	if !ok {
		e = errUnknownAlgo
		return
	}
	return alg.parseFunc(size)
}
