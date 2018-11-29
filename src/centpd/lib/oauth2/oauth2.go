package oauth2

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	jwtreq "github.com/dgrijalva/jwt-go/request"

	"centpd/lib/mail/form"
	ib0 "centpd/lib/webib0"
)

type UserProvider interface {
	UsrLogin(usr, pass string) (attrs map[string]interface{}, err error)
}

type IBOAuth2 struct {
	ib0.IBWebPostProvider
	signKey []byte
	usrprov UserProvider
}

func NewOAuth2Checker(
	wpp ib0.IBWebPostProvider, key []byte, usrprov UserProvider) (*IBOAuth2) {

	return &IBOAuth2{
		IBWebPostProvider: wpp,
		signKey: key,
		usrprov: usrprov,
	}
}

var methodWeUse = jwt.SigningMethodHS256

var _ ib0.IBWebPostProvider = (*IBOAuth2)(nil)

func (s *IBOAuth2) Login(
	r *http.Request, usr, pass string) (tok string, err error, code int) {

	attrs, err := s.usrprov.UsrLogin(usr, pass)
	if err != nil {
		err = fmt.Errorf("login failure: %v", err)
		code = 401
		return
	}

	token := jwt.New(methodWeUse)
	claims := token.Claims.(jwt.MapClaims)
	for k, v := range attrs {
		claims[k] = v
	}

	//claims["admin"] = true
	//claims["name"] = "Ado Kukic"

	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	tokenString, e := token.SignedString(s.signKey)
	if e != nil {
		panic(fmt.Errorf("err from token.SignedString: %v", e))
	}

	// assign
	tok = tokenString
	return
}

func isStillValid(claims jwt.MapClaims) (stillvalid bool, okaytok bool) {
	expclaim, ok := claims["exp"]
	if !ok {
		return false, false
	}
	timetoexp, ok := expclaim.(int64)
	if !ok {
		return false, false
	}
	stillvalid = time.Now().Unix() < timetoexp
	return
}

func (s *IBOAuth2) validateOAuth2(
	r *http.Request) (claims jwt.MapClaims, err error, code int) {

	tok, err := jwtreq.ParseFromRequest(
		r, jwtreq.OAuth2Extractor, jwt.Keyfunc(
			func(token *jwt.Token) (interface{}, error) {
				m, ok := token.Method.(*jwt.SigningMethodHMAC)
				if !ok || m != methodWeUse {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return s.signKey, nil
			}))
	if err != nil {
		err = fmt.Errorf("failed parsing token: %v", err)
		code = 401
		return
	}
	if !tok.Valid {
		err = errors.New("token invalid")
		code = 401
		return
	}
	claims = tok.Claims.(jwt.MapClaims)
	stillvalid, isvalid := isStillValid(claims)
	if !isvalid {
		err = errors.New("token invalid")
		code = 401
		return
	}
	if !stillvalid {
		err = errors.New("token expired")
		code = 401
		return
	}
	
	return
}

func isAdmin(claims jwt.MapClaims) bool {
	return claims["admin"].(bool)
}

func (s *IBOAuth2) IBPostNewBoard(
	r *http.Request, bi ib0.IBNewBoardInfo) (err error, code int) {

	var claims jwt.MapClaims
	claims, err, code = s.validateOAuth2(r)
	if err != nil {
		return
	}
	if !isAdmin(claims) {
		return errors.New("admin privilege needed"), 401
	}
	return s.IBWebPostProvider.IBPostNewBoard(r, bi)
}

func (s *IBOAuth2) IBPostNewThread(
	r *http.Request, f form.Form, board string) (
	rInfo ib0.IBPostedInfo, err error, code int) {

	_, err, code = s.validateOAuth2(r)
	if err != nil {
		return
	}
	return s.IBWebPostProvider.IBPostNewThread(r, f, board)
}

func (s *IBOAuth2) IBPostNewReply(
	r *http.Request, f form.Form, board, thread string) (
	rInfo ib0.IBPostedInfo, err error, code int) {

	_, err, code = s.validateOAuth2(r)
	if err != nil {
		return
	}
	return s.IBWebPostProvider.IBPostNewReply(r, f, board, thread)
}

func (s *IBOAuth2) IBDeleteBoard(
	r *http.Request, board string) (err error, code int) {

	var claims jwt.MapClaims
	claims, err, code = s.validateOAuth2(r)
	if err != nil {
		return
	}
	if !isAdmin(claims) {
		return errors.New("admin privilege needed"), 401
	}
	return s.IBWebPostProvider.IBDeleteBoard(r, board)
}

func (s *IBOAuth2) IBDeletePost(
	r *http.Request, board, post string) (err error, code int) {

	var claims jwt.MapClaims
	claims, err, code = s.validateOAuth2(r)
	if err != nil {
		return
	}
	if !isAdmin(claims) {
		return errors.New("admin privilege needed"), 401
	}
	return s.IBWebPostProvider.IBDeletePost(r, board, post)
}
