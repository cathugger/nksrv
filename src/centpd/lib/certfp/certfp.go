package certfp

import (
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/sha3"
)

type Selector int

const (
	SelectorFull   = Selector(0)
	SelectorPubKey = Selector(1)
	SelectorDigest = Selector(2)
)

type MatchingType int

const (
	MatchingTypeIdentity MatchingType = iota
	MatchingTypeSHA2_224
	MatchingTypeSHA2_256
	MatchingTypeSHA2_384
	MatchingTypeSHA2_512
	MatchingTypeSHA3_224
	MatchingTypeSHA3_256
	MatchingTypeSHA3_384
	MatchingTypeSHA3_512
	MatchingTypeSHAKE128
	MatchingTypeSHAKE256
	MatchingTypeBLAKE2s
	MatchingTypeBLAKE2b
	_MatchingTypeMax
)

var MatchingTypeStr = [_MatchingTypeMax]string{
	"ident",
	"sha224",
	"sha256",
	"sha384",
	"sha512",
	"sha3-224",
	"sha3-256",
	"sha3-384",
	"sha3-512",
	"shake128",
	"shake256",
	"blake2s",
	"blake2b",
}

var errUnknown = errors.New("unknown fingerprint type")
var errBadSize = errors.New("unsupported fingerprint length")

func ParseMatchingType(s string) (mt MatchingType, err error) {
	ts := strings.ToLower(s)
	switch ts {
	case "ident", "id", "identity":
		mt = MatchingTypeIdentity

	case "sha2-224", "sha224":
		mt = MatchingTypeSHA2_224
	case "sha2-256", "sha256":
		mt = MatchingTypeSHA2_256
	case "sha2-384", "sha384":
		mt = MatchingTypeSHA2_384
	case "sha2-512", "sha512":
		mt = MatchingTypeSHA2_512

	case "sha3-224":
		mt = MatchingTypeSHA3_224
	case "sha3-256":
		mt = MatchingTypeSHA3_256
	case "sha3-384":
		mt = MatchingTypeSHA3_384
	case "sha3-512":
		mt = MatchingTypeSHA3_512

	case "shake128":
		mt = MatchingTypeSHAKE128
	case "shake256":
		mt = MatchingTypeSHAKE256

	case "blake2s", "blake2s256":
		mt = MatchingTypeBLAKE2s
	case "blake2b", "blake2b512", "blake2":
		mt = MatchingTypeBLAKE2b

	default:
		err = errUnknown
	}
	return
}

func ParseCertFP(s string) (mt MatchingType, data []byte, err error) {
	i := strings.IndexByte(s, ':')
	if i < 0 {
		err = errUnknown
		return
	}

	mt, err = ParseMatchingType(s[:i])
	if err != nil {
		return
	}

	purehex := strings.Replace(s[i+1:], ":", "", -1)
	data, err = hex.DecodeString(purehex)
	if err != nil {
		return
	}
	switch mt {
	case MatchingTypeSHA2_224, MatchingTypeSHA3_224:
		if len(data) != 28 {
			err = errBadSize
		}
	case MatchingTypeSHA2_256, MatchingTypeSHA3_256,
		MatchingTypeSHAKE128, MatchingTypeBLAKE2s:
		if len(data) != 32 {
			err = errBadSize
		}
	case MatchingTypeSHA2_384, MatchingTypeSHA3_384:
		if len(data) != 48 {
			err = errBadSize
		}
	case MatchingTypeSHA2_512, MatchingTypeSHA3_512,
		MatchingTypeSHAKE256, MatchingTypeBLAKE2b:
		if len(data) != 64 {
			err = errBadSize
		}
	}
	return
}

func MakeFingerprint(
	cert *x509.Certificate, selector Selector, mt MatchingType) []byte {

	var data []byte
	switch selector {
	case SelectorFull:
		data = cert.Raw
	case SelectorPubKey:
		data = cert.RawSubjectPublicKeyInfo
	case SelectorDigest:
		data = cert.RawTBSCertificate
	default:
		panic("unknown Selector")
	}

	switch mt {
	case MatchingTypeIdentity:
		return data

	case MatchingTypeSHA2_224:
		h := sha256.Sum224(data)
		return h[:]

	case MatchingTypeSHA2_256:
		h := sha256.Sum256(data)
		return h[:]

	case MatchingTypeSHA2_384:
		h := sha512.Sum384(data)
		return h[:]

	case MatchingTypeSHA2_512:
		h := sha512.Sum512(data)
		return h[:]

	case MatchingTypeSHA3_224:
		h := sha3.Sum224(data)
		return h[:]

	case MatchingTypeSHA3_256:
		h := sha3.Sum256(data)
		return h[:]

	case MatchingTypeSHA3_384:
		h := sha3.Sum384(data)
		return h[:]

	case MatchingTypeSHA3_512:
		h := sha3.Sum512(data)
		return h[:]

	case MatchingTypeSHAKE128:
		var h [32]byte
		sha3.ShakeSum128(h[:], data)
		return h[:]

	case MatchingTypeSHAKE256:
		var h [64]byte
		sha3.ShakeSum256(h[:], data)
		return h[:]

	case MatchingTypeBLAKE2s:
		h := blake2s.Sum256(data)
		return h[:]

	case MatchingTypeBLAKE2b:
		h := blake2b.Sum512(data)
		return h[:]

	default:
		panic("unknown MatchingType")
	}
}

func FingerprintString(mt MatchingType, data []byte) string {
	return MatchingTypeStr[mt] + ":" + hex.EncodeToString(data)
}
