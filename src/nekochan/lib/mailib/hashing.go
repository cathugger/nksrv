package mailib

import (
	crand "crypto/rand"
	"crypto/sha1"
	"encoding/hex"

	ht "nekochan/lib/hashtools"
)

func NewRandomMessageID(t int64, name string) FullMsgIDStr {
	// TAI64
	var b [8]byte
	u := uint64(t) + 4611686018427387914
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
	var r [12]byte
	crand.Read(r[:])

	return FullMsgIDStr("<" + ht.SBase64Enc.EncodeToString(b[:]) + "." +
		ht.SBase64Enc.EncodeToString(r[:]) + "@" + name + ">")
}

// TODO: more algos
func HashPostID_SHA1(coremsgid FullMsgIDStr) string {
	b := sha1.Sum(unsafeStrToBytes(string(coremsgid)))
	return hex.EncodeToString(b[:])
}
