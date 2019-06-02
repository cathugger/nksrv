package captcha

import (
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

type Answer string
type Distortion []byte

/*
 * layout for key:
 * 8[keyid]|24[nonce]|?[encrypted data]|16[poly1305]
 * 8+24+16 = 48
 * leyout for data:
 * 1[type]|8[expiredate]|1[l]|6[keydata]|16[seed]
 * 1+8+1+6+16 = 32
 * 48 + 32 = 80
 * 80*8/5=128
 */

const MACOverhead = 16
const MaxChalLen = 12
const KeyDataLen = 32
const EncKeyLen = 80
const DefaultChalLen = 8
const DefaultExpireSecs = 60 * 15 // 15 mins

func EncryptKey(kek cipher.AEAD, keyid uint64, k []byte, t int64) (ek []byte) {
	if kek.Overhead() != MACOverhead {
		panic("bad kek.Overhead()")
	}

	b := make([]byte, 8+24+len(k)+MACOverhead)

	binary.BigEndian.PutUint64(b[0:8], keyid)

	binary.BigEndian.PutUint32(b[8:8+4], uint32(uint64(t)>>1))

	_, err := crand.Read(b[8+4 : 8+24])
	if err != nil {
		panic("crand read: " + err.Error())
	}

	kek.Seal(b[8+24:8+24], b[8:8+24], k, b[0:8])

	return b
}

func ParseKEK(kek []byte) cipher.AEAD {
	c, err := chacha20poly1305.NewX(kek)
	if err != nil {
		panic("chacha20poly1305.NewX err: " + err.Error())
	}
	return c
}

func RandomKEK() (id uint64, kek []byte) {
	var b [chacha20poly1305.KeySize + 8]byte

	_, err := crand.Read(b[:])
	if err != nil {
		panic("crand.Read err: " + err.Error())
	}

	kek = b[:chacha20poly1305.KeySize]
	id = binary.LittleEndian.Uint64(b[chacha20poly1305.KeySize:]) & 0x7FffFFffFFffFFff
	return
}

func DecryptKey(kek cipher.AEAD, ek []byte) (k []byte, err error) {
	if len(ek) != EncKeyLen {
		err = errors.New("invalid key")
	}

	k, err = kek.Open(ek[8+24:8+24], ek[8:8+24], ek[8+24:], ek[0:8])
	if err != nil {
		err = errors.New("key verification failed")
		return
	}

	return
}

func PackKeyData(
	k []byte, typ uint8, exp int64, chal []byte, seed [16]byte) {

	k[0] = typ

	binary.BigEndian.PutUint64(k[1:1+8], uint64(exp)+0x4000000000000000)

	k[1+8] = byte(len(chal))

	if len(chal) > MaxChalLen {
		panic("challenge too long")
	}
	_, err := packBCD(k[1+8+1:1+8+1], chal)
	if err != nil {
		panic("packBCD err: " + err.Error())
	}

	copy(k[1+8+1+6:], seed[:])
}

func UnpackKeyData(
	k []byte) (typ uint8, exp int64, chal []byte, seed [16]byte, err error) {

	typ = k[0]

	exp = int64(binary.BigEndian.Uint64(k[1:1+8]) - 0x4000000000000000)

	challen := k[1+8]
	if challen > MaxChalLen {
		err = errors.New("bad challenge length")
	}

	chal, err = unpackBCD(nil, k[1+8+1:1+8+1+6], int(challen))
	if err != nil {
		return
	}

	copy(seed[:], k[1+8+1+6:])

	return
}

func RandomChallenge(length int) (chal []byte, seed [16]byte) {
	if length <= 0 {
		length = DefaultChalLen
	} else if length > MaxChalLen {
		panic("challenge len too large")
	}

	chal = RandomDigits(length)

	_, err := crand.Read(seed[:])
	if err != nil {
		panic("crand read: " + err.Error())
	}

	return
}

func EncryptChallenge(
	kek cipher.AEAD, keyid uint64,
	typ uint8, expp int64, chal []byte, seed [16]byte) []byte {

	if expp <= 0 {
		expp = DefaultExpireSecs
	}

	t := time.Now().Unix()

	var kd [KeyDataLen]byte
	PackKeyData(kd[:], typ, t+expp, chal, seed)

	return EncryptKey(kek, keyid, kd[:], t)
}
