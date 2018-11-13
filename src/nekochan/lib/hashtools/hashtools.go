package hashtools

import (
	"encoding/base32"
	"encoding/base64"
	"io"
	"os"

	"golang.org/x/crypto/blake2s"
)

// lowecase base32 set which preserves sorting order
var LowerBase32HexSet = "0123456789abcdefghijklmnopqrstuv"

// encoding for that
var LowerBase32HexEnc = base32.
	NewEncoding(LowerBase32HexSet).
	WithPadding(base32.NoPadding)

// custom base64 set (preserves sort order)
var SBase64Set = "-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"_abcdefghijklmnopqrstuvwxyz"

// encoding for that
var SBase64Enc = base64.
	NewEncoding(SBase64Set).
	WithPadding(base64.NoPadding)

// MakeFileHash returns textural representation of file hash.
// It expects file to be seeked at 0.
func MakeFileHash(f *os.File) (s string, e error) {
	h, e := blake2s.New256([]byte(nil))
	if e != nil {
		return
	}

	_, e = io.Copy(h, f)
	if e != nil {
		return
	}

	var b [32]byte
	sum := h.Sum(b[:0])
	s = LowerBase32HexEnc.EncodeToString(sum)

	return
}
