package mailib

import (
	crand "crypto/rand"
	"crypto/sha1"
	"encoding/hex"

	ht "nksrv/lib/hashtools"
)

func NewRandomMessageID(t int64, name string) TFullMsgIDStr {
	// TAI64 format kinda
	var b [8]byte
	u := uint64(0x4000000000000000 + t)
	b[7] = byte(u)
	u >>= 8
	b[6] = byte(u)
	u >>= 8
	b[5] = byte(u)
	u >>= 8
	b[4] = byte(u)
	u >>= 8
	b[3] = byte(u)
	u >>= 8
	b[2] = byte(u)
	u >>= 8
	b[1] = byte(u)
	u >>= 8
	b[0] = byte(u)

	// random
	var r [10]byte
	crand.Read(r[:])

	// non-recent nntpchan (fixed in 2d3c304c81b5) can't handle base64url...
	return TFullMsgIDStr("<" +
		ht.LowerBase32HexEnc.EncodeToString(b[:]) + "." +
		ht.LowerBase32HexEnc.EncodeToString(r[:]) + "@" + name + ">")
}

// TODO: more algos
func HashPostID_SHA1(coremsgid TFullMsgIDStr) string {
	b := sha1.Sum(unsafeStrToBytes(string(coremsgid)))
	return hex.EncodeToString(b[:])
}
