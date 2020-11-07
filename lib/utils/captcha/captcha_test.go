package captcha

import (
	"bytes"
	"testing"
)

func TestCaptcha(t *testing.T) {
	id, kek := RandomKEK()
	pkek := ParseKEK(kek)

	chal, seed := RandomChallenge(0)

	enckey, _, _ := EncryptChallenge(pkek, id, 0, 60*15, chal, seed)

	deckey, err := DecryptKey(pkek, enckey)
	if err != nil {
		t.Errorf("DecryptKey err: %v", err)
		return
	}

	dtyp, dexp, dchal, dseed, err := UnpackKeyData(deckey)
	if err != nil {
		t.Errorf("UnpackKeyData err: %v", err)
		return
	}

	if dtyp != 0 {
		t.Errorf("dtyp expected 0 got %d", dtyp)
	}

	_ = dexp

	if !bytes.Equal(chal, dchal) {
		t.Errorf("dchal expected %v got %v", chal, dchal)
	}

	if !bytes.Equal(seed[:], dseed[:]) {
		t.Errorf("dseed expected %v got %v", seed, dseed)
	}
}
