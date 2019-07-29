package hashtools

import (
	"encoding/base32"
	"encoding/base64"
	"io"

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

// MakeFileHash returns textural representation of file hash.
// It expects file to be seeked at 0.
func MakeFileHash(r io.Reader) (s string, hs string, e error) {
	h, e := blake2b.New(28, nil)
	if e != nil {
		return
	}

	_, e = io.Copy(h, r)
	if e != nil {
		return
	}

	var b [28]byte
	sum := h.Sum(b[:0])
	s = LowerBase32Enc.EncodeToString(sum)
	hs = "b2b"

	return
}
