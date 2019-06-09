package webcaptcha

import (
	"bytes"
	"crypto/cipher"
	"encoding/base32"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/text/unicode/norm"

	"centpd/lib/captcha"
	"centpd/lib/captchastore"
	ib0 "centpd/lib/webib0"
)

type captchaKey struct {
	kek      cipher.AEAD
	disabled bool
}

type WebCaptcha struct {
	keys       map[uint64]captchaKey
	store      captchastore.CaptchaStore
	usecookies bool
}

var errInvalidMissing = errors.New("invalid submission: doesn't include captcha fields")

var keyenc = base32.StdEncoding.WithPadding(base32.NoPadding)

func NewWebCaptcha(store captchastore.CaptchaStore, usecookies bool) (*WebCaptcha, error) {
	keks, err := store.LoadKEKs(captcha.RandomKEK)
	if err != nil {
		return nil, err
	}
	keys := make(map[uint64]captchaKey)
	for i := range keks {
		pkek := captcha.ParseKEK(keks[i].KEK)
		keys[keks[i].ID] = captchaKey{
			kek:      pkek,
			disabled: keks[i].Disabled,
		}
	}
	return &WebCaptcha{
		keys:       keys,
		store:      store,
		usecookies: usecookies,
	}, nil
}

func text2fbcd(txt string) ([]byte, error) {
	// for captcha use only ascii text, so it makes sense to correct any ascii-ish stuff
	txt = norm.NFKC.String(txt)
	b := make([]byte, len(txt))
	for i := 0; i < len(txt); i++ {
		c := txt[i]
		if c >= '0' && c <= '9' {
			b[i] = c - '0'
		} else {
			return nil, errors.New("invalid character")
		}
	}
	return b, nil
}

func (wc *WebCaptcha) CheckCaptcha(r *http.Request, fields map[string][]string) (error, int) {
	fcaptchakey := ib0.IBWebFormTextCaptchaKey
	fcaptchaans := ib0.IBWebFormTextCaptchaAns

	var xfcaptchakey, xfcaptchaans string
	if !wc.usecookies {
		if len(fields[fcaptchakey]) != 1 || len(fields[fcaptchaans]) != 1 {
			return errInvalidMissing, http.StatusBadRequest
		}
		xfcaptchakey = fields[fcaptchakey][0]
		xfcaptchaans = fields[fcaptchaans][0]
	} else {
		if len(fields[fcaptchaans]) != 1 {
			return errInvalidMissing, http.StatusBadRequest
		}
		xfcaptchaans = fields[fcaptchaans][0]

		c, e := r.Cookie("captchakey")
		if e != nil {
			return errors.New("invalid submission: missing captchakey cookie"), http.StatusBadRequest
		}
		xfcaptchakey = c.Value
	}

	ek, e := keyenc.DecodeString(xfcaptchakey)
	if e != nil {
		return fmt.Errorf("invalid captcha key: %v", e), http.StatusBadRequest
	}

	id, e := captcha.UnpackKeyID(ek)
	if e != nil {
		return fmt.Errorf("invalid captcha key: %v", e), http.StatusBadRequest
	}

	kek, ok := wc.keys[id]
	if !ok {
		return errors.New("captcha key id not known"), http.StatusUnauthorized
	}
	if kek.disabled {
		return errors.New("captcha key id disabled"), http.StatusUnauthorized
	}
	dk, e := captcha.DecryptKey(kek.kek, ek)
	if e != nil {
		return fmt.Errorf("invalid captcha key: %v", e), http.StatusBadRequest
	}
	typ, exp, chal, _, e := captcha.UnpackKeyData(dk)
	if e != nil {
		return fmt.Errorf("invalid captcha key: %v", e), http.StatusBadRequest
	}
	nowt := time.Now().Unix()
	if nowt >= exp {
		return errors.New("expired captcha key"), http.StatusUnauthorized
	}
	if typ != 0 {
		return errors.New("invalid captcha key: unsupported type"), http.StatusBadRequest
	}
	bans, e := text2fbcd(xfcaptchaans)
	if e != nil {
		return fmt.Errorf("invalid captcha answer: %v", e), http.StatusUnauthorized
	}
	if !bytes.Equal(chal, bans) {
		return errors.New("incorrect captcha answer"), http.StatusUnauthorized
	}
	fresh, e := wc.store.StoreSolved(dk, exp, nowt)
	if e != nil {
		return fmt.Errorf("captcha store err: %v", e), http.StatusInternalServerError
	}
	if !fresh {
		return errors.New("captcha key is already used up"), http.StatusUnauthorized
	}
	// all good
	return nil, 0
}
