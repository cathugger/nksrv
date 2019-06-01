package captcha

import (
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
)

type Answer string
type Distortion []byte

/*
 * layout for key:
 * 8[keyid]|24[nonce]|?[encrypted data]
 * leyout for data:
 * 2[type]|8[expiredate]|1[x]|x[keydata]|?[random]
 */

func EncryptKey(kek cipher.AEAD, keyid uint64, k []byte, t int64) (ek []byte) {
	b := make([]byte, 8+24+len(k)+kek.Overhead())

	binary.BigEndian.PutUint64(b[0:8], keyid)

	binary.BigEndian.PutUint32(b[8:8+4], uint32(uint64(t)>>1))

	_, err := crand.Read(b[8+4 : 8+24])
	if err != nil {
		panic("crand read: " + err.Error())
	}

	kek.Seal(b[8+24:8+24], b[8:8+24], k, b[0:8])

	return b
}

func DecryptKey(kek cipher.AEAD, ek []byte) (k []byte, err error) {
	if len(ek) <= 8+24+kek.Overhead() {
		err = errors.New("invalid key")
	}

	k, err = kek.Open(ek[8+24:8+24], ek[8:8+24], ek[8+24:], ek[0:8])
	if err != nil {
		err = errors.New("key verification failed")
		return
	}

	return
}
