package hashtools

import (
	"encoding/base32"
	"encoding/base64"
	"io"
	"math/big"

	"golang.org/x/crypto/blake2b"
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

const (
	ht_SHA2_224    = 1
	ht_BLAKE2b_224 = 2
)

// MakeFileHash returns textural representation of file hash for use in filename.
// It expects file to be seeked at 0.
func MakeFileHash(r io.Reader) (s string, e error) {
	const hlen = 28
	const slen = 48 // technically 44, but wont hurt to have a bit more
	var b [slen]byte

	b[0] = ht_BLAKE2b_224
	// hash
	h, e := blake2b.New(hlen, nil)
	if e != nil {
		return
	}
	_, e = io.Copy(h, r)
	if e != nil {
		return
	}
	h.Sum(b[1:][:0])

	// convert to base36 number and print it
	var x big.Int
	x.SetBytes(b[:1+hlen])
	xb := x.Append(b[:0], 36)

	// flip (we want front bits to be more variable)
	for i, j := 0, len(xb)-1; i < j; i, j = i+1, j-1 {
		xb[i], xb[j] = xb[j], xb[i]
	}

	// it's ready
	s = string(xb)

	return
}
