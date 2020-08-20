package mailibsign

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/ed25519"

	"nksrv/lib/mail"
	au "nksrv/lib/utils/asciiutils"
)

func fromhex(str string) ([]byte, error) {
	return hex.DecodeString(str)
}

func tohex(b []byte) string {
	return fmt.Sprintf("%X", b)
}

type InnerWriter interface {
	hash.Hash
}

type VerifyResult struct {
	// TODO
	PubKey string
}

type Verifier interface {
	Verify(iw InnerWriter) VerifyResult
}

func PrepareVerifier(
	H mail.HeaderMap, ct_t string, ct_par map[string]string, innermsg bool) (
	ver Verifier, iow InnerWriter) {

	if innermsg {
		keyhdr := au.TrimWSString(H.GetFirst("X-PubKey-Ed25519"))
		if keyhdr != "" {

			pk, e := hex.DecodeString(keyhdr)
			if e == nil && len(pk) == ed25519.PublicKeySize {

				var sighdr string
				var sig []byte

				sighdr = au.TrimWSString(H.GetFirst("X-Signature-Ed25519-SHA512"))
				if sighdr != "" {

					sig, e = hex.DecodeString(sighdr)
					if e == nil && len(sig) == ed25519.SignatureSize {

						ver = VerifierEd25519{
							pk:  ed25519.PublicKey(pk),
							sig: sig,
						}

						iow = InnerWriter(sha512.New())

						return
					}
				}

				sighdr = au.TrimWSString(H.GetFirst("X-Signature-Ed25519-BLAKE2b"))
				if sighdr != "" {

					sig, e = hex.DecodeString(sighdr)
					if e == nil && len(sig) == ed25519.SignatureSize {

						ver = VerifierEd25519{
							pk:  ed25519.PublicKey(pk),
							sig: sig,
						}

						h, _ := blake2b.New512(nil)
						iow = InnerWriter(h)

						return
					}
				}
			}
		}
	}

	return
}

type VerifierEd25519 struct {
	pk  ed25519.PublicKey
	sig []byte
}

func (v VerifierEd25519) Verify(iw InnerWriter) (res VerifyResult) {
	var sumbuf [64]byte
	sum := iw.Sum(sumbuf[:0])
	//fmt.Printf("sig ver hash %X pubkey %X signature %X\n", sum, v.pk, v.sig)
	if ed25519.Verify(v.pk, sum, v.sig) {
		res.PubKey = tohex(v.pk)
		//fmt.Printf("sig ver okay\n")
	} else {
		//fmt.Printf("sig ver fail\n")
	}
	return
}
