package webcaptcha

import (
	"bytes"
	"crypto/cipher"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/text/unicode/norm"

	"centpd/lib/captcha"
	"centpd/lib/captchastore"
	"centpd/lib/hashtools"
	ib0 "centpd/lib/webib0"
)

type captchaKey struct {
	kek      cipher.AEAD
	disabled bool
}

type WebCaptcha struct {
	keys          map[uint64]captchaKey
	prim          uint64
	store         captchastore.CaptchaStore
	usecookies    bool
	length        int
	validduration int64
}

var errInvalidMissing = errors.New(
	"invalid submission: doesn't include captcha fields")

var keyenc = hashtools.LowerBase32Enc

func NewWebCaptcha(
	store captchastore.CaptchaStore, usecookies bool) (*WebCaptcha, error) {

	keks, err := store.LoadKEKs(captcha.RandomKEK)
	if err != nil {
		return nil, err
	}
	keys := make(map[uint64]captchaKey)
	prim := int64(-1)
	for i := range keks {
		pkek := captcha.ParseKEK(keks[i].KEK)
		id := keks[i].ID
		disabled := keks[i].Disabled
		keys[id] = captchaKey{
			kek:      pkek,
			disabled: disabled,
		}
		if !disabled && prim < 0 {
			prim = int64(id)
		}
	}
	return &WebCaptcha{
		keys:       keys,
		prim:       uint64(prim),
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

func (wc *WebCaptcha) unpackAndValidateKey(
	str string, nowt int64) (
	dk []byte, typ byte, exp int64,
	chal []byte, seed [16]byte, err error, code int) {

	ek, err := keyenc.DecodeString(str)
	if err != nil {
		err, code = fmt.Errorf("invalid captcha key: %v", err),
			http.StatusBadRequest
		return
	}
	id, err := captcha.UnpackKeyID(ek)
	if err != nil {
		err, code = fmt.Errorf("invalid captcha key: %v", err),
			http.StatusBadRequest
		return
	}
	kek, ok := wc.keys[id]
	if !ok {
		err, code = errors.New("captcha key id not known"),
			http.StatusUnauthorized
		return
	}
	if kek.disabled {
		err, code = errors.New("captcha key id disabled"),
			http.StatusUnauthorized
		return
	}
	dk, err = captcha.DecryptKey(kek.kek, ek)
	if err != nil {
		err, code = fmt.Errorf("invalid captcha key: %v", err),
			http.StatusBadRequest
		return
	}
	typ, exp, chal, seed, err = captcha.UnpackKeyData(dk)
	if err != nil {
		err, code = fmt.Errorf("invalid captcha key: %v", err),
			http.StatusBadRequest
		return
	}
	if nowt >= exp {
		err, code = errors.New("expired captcha key"), http.StatusUnauthorized
		return
	}
	if typ != 0 {
		err, code = errors.New("invalid captcha key: unsupported type"),
			http.StatusBadRequest
		return
	}
	// oki so far
	return
}

func (wc *WebCaptcha) CheckCaptcha(
	r *http.Request, fields map[string][]string) (err error, code int) {

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

		c, err := r.Cookie(fcaptchakey)
		if err != nil {
			return errors.New("invalid submission: missing captchakey cookie"),
				http.StatusBadRequest
		}
		xfcaptchakey = c.Value
	}

	nowt := time.Now().Unix()
	dk, _, exp, chal, _, err, code :=
		wc.unpackAndValidateKey(xfcaptchakey, nowt)
	if err != nil {
		return
	}
	b_ans, err := text2fbcd(xfcaptchaans)
	if err != nil {
		return fmt.Errorf("invalid captcha answer: %v", err),
			http.StatusUnauthorized
	}
	if !bytes.Equal(chal, b_ans) {
		return errors.New("incorrect captcha answer"), http.StatusUnauthorized
	}
	fresh, err := wc.store.StoreSolved(dk, exp, nowt)
	if err != nil {
		return fmt.Errorf("captcha store err: %v", err),
			http.StatusInternalServerError
	}
	if !fresh {
		return errors.New("captcha key is already used up"),
			http.StatusUnauthorized
	}
	// all good
	return
}

func (wc *WebCaptcha) ServeCaptchaPNG(
	w http.ResponseWriter, key string, width, height int) (
	err error, code int) {

	var chal []byte
	var seed [16]byte

	if !wc.usecookies {
		_, _, _, chal, seed, err, code =
			wc.unpackAndValidateKey(key, time.Now().Unix())
		if err != nil {
			return
		}
	} else {
		chal, seed = captcha.RandomChallenge(wc.length)
		ek, _, _ := captcha.EncryptChallenge(
			wc.keys[wc.prim].kek, wc.prim, 0, wc.validduration, chal, seed)
		// XXX think of better attribs for cookie
		http.SetCookie(w, &http.Cookie{
			Name:  ib0.IBWebFormTextCaptchaKey,
			Value: keyenc.EncodeToString(ek),
		})
	}

	img := captcha.NewImage(chal, seed, width, height)
	img.WritePNG(w) // XXX pre-buffer?
	return
}

func (wc *WebCaptcha) NewKey() string {
	chal, seed := captcha.RandomChallenge(wc.length)
	ek, _, _ := captcha.EncryptChallenge(
		wc.keys[wc.prim].kek, wc.prim, 0, wc.validduration, chal, seed)
	return keyenc.EncodeToString(ek)
}
