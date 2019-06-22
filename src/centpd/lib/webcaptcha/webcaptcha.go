package webcaptcha

import (
	"bytes"
	"crypto/cipher"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	keks          map[uint64]captchaKey
	prim          uint64
	primkek       cipher.AEAD
	store         captchastore.CaptchaStore
	length        int
	validduration int64

	UseCookies bool
}

var errInvalidMissing = errors.New(
	"invalid submission: doesn't include captcha fields")

var keyenc = hashtools.LowerBase32Enc

func NewWebCaptcha(
	store captchastore.CaptchaStore, usecookies bool) (*WebCaptcha, error) {

	lkeks, err := store.LoadKEKs(captcha.RandomKEK)
	if err != nil {
		return nil, err
	}
	keks := make(map[uint64]captchaKey)
	prim := int64(-1)
	var primkek cipher.AEAD
	for i := range lkeks {
		pkek := captcha.ParseKEK(lkeks[i].KEK)
		id := lkeks[i].ID
		disabled := lkeks[i].Disabled
		keks[id] = captchaKey{
			kek:      pkek,
			disabled: disabled,
		}
		if !disabled && prim < 0 {
			prim = int64(id)
			primkek = pkek
		}
	}
	return &WebCaptcha{
		keks:       keks,
		prim:       uint64(prim),
		primkek:    primkek,
		store:      store,
		UseCookies: usecookies,
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
	if len(ek) == 0 {
		err, code = errors.New("no captcha key provided"), http.StatusBadRequest
		return
	}
	id, err := captcha.UnpackKeyID(ek)
	if err != nil {
		err, code = fmt.Errorf("invalid captcha key: %v", err),
			http.StatusBadRequest
		return
	}
	kek, ok := wc.keks[id]
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

const (
	fcaptchakey = ib0.IBWebFormTextCaptchaKey
	fcaptchaans = ib0.IBWebFormTextCaptchaAns
)

func (wc *WebCaptcha) TextFields() []string {
	if !wc.UseCookies {
		return []string{fcaptchakey, fcaptchaans}
	} else {
		return []string{fcaptchaans}
	}
}

func (wc *WebCaptcha) CheckCaptcha(
	r *http.Request, fields map[string][]string) (err error, code int) {

	var xfcaptchakey, xfcaptchaans string
	if !wc.UseCookies {
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
	w http.ResponseWriter, r *http.Request, key string, width, height int) (
	err error, code int) {

	var chal []byte
	var seed [16]byte

	if !wc.UseCookies {
		_, _, _, chal, seed, err, code =
			wc.unpackAndValidateKey(key, time.Now().Unix())
		if err != nil {
			return
		}
	} else {
		chal, seed = captcha.RandomChallenge(wc.length)
		ek, _, _ := captcha.EncryptChallenge(
			wc.primkek, wc.prim, 0, wc.validduration, chal, seed)
		// XXX think of better attribs for cookie
		http.SetCookie(w, &http.Cookie{
			Name:  ib0.IBWebFormTextCaptchaKey,
			Value: keyenc.EncodeToString(ek),
		})
	}

	// TODO sync.Pool if there's need
	img := captcha.NewImage(chal, seed, width, height)
	var b bytes.Buffer
	img.WritePNG(&b)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
	w.Write(b.Bytes())
	return
}

func (wc *WebCaptcha) NewKey() string {
	chal, seed := captcha.RandomChallenge(wc.length)
	ek, _, _ := captcha.EncryptChallenge(
		wc.primkek, wc.prim, 0, wc.validduration, chal, seed)
	return keyenc.EncodeToString(ek)
}
