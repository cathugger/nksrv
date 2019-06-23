package tmplrenderer

import (
	"net/http"

	ib0 "centpd/lib/webib0"
)

func (tr *TmplRenderer) DressNewBoardResult(
	w http.ResponseWriter, bname string, err error, code int) {

	l := &struct {
		S bool   // success
		B string // board name
		E error
		C int
		N *NodeInfo
		R *TmplRenderer
	}{
		S: err == nil,
		B: bname,
		E: err,
		C: code,
		N: &tr.ni,
		R: tr,
	}
	if err == nil {
		tr.outTmplR(w, rtmplCreatedBoard, 200, l)
	} else {
		tr.outTmplR(w, rtmplCreatedBoardErr, code, l)
	}
}

func (tr *TmplRenderer) DressPostResult(
	w http.ResponseWriter, pi ib0.IBPostedInfo, newthread bool,
	err error, code int) {

	l := &struct {
		D ib0.IBPostedInfo
		E error
		C int
		N *NodeInfo
		R *TmplRenderer
	}{
		D: pi,
		E: err,
		C: code,
		N: &tr.ni,
		R: tr,
	}
	if newthread {
		if err == nil {
			tr.outTmplR(w, rtmplCreatedThread, 200, l)
		} else {
			tr.outTmplR(w, rtmplCreatedThreadErr, code, l)
		}
	} else {
		if err == nil {
			tr.outTmplR(w, rtmplCreatedPost, 200, l)
		} else {
			tr.outTmplR(w, rtmplCreatedPostErr, code, l)
		}
	}
}

// XXX kinda bad place to put this
func (tr *TmplRenderer) WebCaptchaInclude(
	w http.ResponseWriter, r *http.Request) {

	// XXX IP whitelist? though that could be server-wise at that point

	// make document containing key uncacheable
	w.Header().Set(
		"Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	tr.outTmplR(w, rtmplCaptchaInclude, 200, tr.wc.NewKey())
}
