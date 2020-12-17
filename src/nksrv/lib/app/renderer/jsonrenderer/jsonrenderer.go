package jsonrenderer

import (
	//. "nksrv/lib/utils/logx"
	"encoding/json"
	"net/http"

	"nksrv/lib/app/renderer"
	ib0 "nksrv/lib/app/webib0"
)

var _ renderer.Renderer = (*JSONRenderer)(nil)

type JSONRenderer struct {
	p      ib0.IBProvider
	indent string
}

type Config struct {
	Indent string
}

func NewJSONRenderer(prov ib0.IBProvider, cfg Config) (*JSONRenderer, error) {
	r := new(JSONRenderer)
	r.p = prov
	r.indent = cfg.Indent
	return r, nil
}

type jsonErrorMsg struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

type jsonError struct {
	Err jsonErrorMsg `json:"error"`
}

func (j *JSONRenderer) prepareEncoder(
	w http.ResponseWriter, code int) *json.Encoder {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if code != 0 {
		w.WriteHeader(code)
	}
	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	e.SetIndent("", j.indent)
	return e
}

func returnError(
	w http.ResponseWriter, e *json.Encoder, err error, code int) {

	w.WriteHeader(code)
	jerr := jsonError{Err: jsonErrorMsg{Code: code, Msg: err.Error()}}
	e.Encode(&jerr)
}

func (j *JSONRenderer) ServeBoardList(
	w http.ResponseWriter, r *http.Request) {

	e := j.prepareEncoder(w, 0)
	var list ib0.IBBoardList
	err, code := j.p.IBGetBoardList(&list)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&list)
}

func (j *JSONRenderer) ServeThreadListPage(
	w http.ResponseWriter, r *http.Request, board string, page uint32) {

	e := j.prepareEncoder(w, 0)
	var pag ib0.IBThreadListPage
	err, code := j.p.IBGetThreadListPage(&pag, board, page)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&pag)
}

func (j *JSONRenderer) ServeOverboardPage(
	w http.ResponseWriter, r *http.Request, page uint32) {

	e := j.prepareEncoder(w, 0)
	var pag ib0.IBOverboardPage
	err, code := j.p.IBGetOverboardPage(&pag, page)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&pag)
}

func (j *JSONRenderer) ServeThreadCatalog(
	w http.ResponseWriter, r *http.Request, board string) {

	e := j.prepareEncoder(w, 0)
	var pag ib0.IBThreadCatalog
	err, code := j.p.IBGetThreadCatalog(&pag, board)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&pag)
}

func (j *JSONRenderer) ServeOverboardCatalog(
	w http.ResponseWriter, r *http.Request) {

	e := j.prepareEncoder(w, 0)
	var pag ib0.IBOverboardCatalog
	err, code := j.p.IBGetOverboardCatalog(&pag)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&pag)
}

func (j *JSONRenderer) ServeThread(
	w http.ResponseWriter, r *http.Request, board, thread string) {

	e := j.prepareEncoder(w, 0)
	var pag ib0.IBThreadPage
	err, code := j.p.IBGetThread(&pag, board, thread)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&pag)
}

func (j *JSONRenderer) DressNewBoardResult(
	w http.ResponseWriter, bname string, err error, code int) {

	e := j.prepareEncoder(w, code)

	r := &struct {
		Success bool   `json:"success"`
		BName   string `json:"bname"`

		jsonErrorMsg
	}{
		Success: err == nil,
		BName:   bname,
	}
	if err != nil {
		r.Msg = err.Error()
		r.Code = code
	}

	e.Encode(r)
}

func (j *JSONRenderer) DressPostResult(
	w http.ResponseWriter, pi ib0.IBPostedInfo, newthread bool,
	err error, code int) {

	e := j.prepareEncoder(w, code)

	ps := &struct {
		Success   bool             `json:"success"`
		NewThread bool             `json:"new_thread"`
		Info      ib0.IBPostedInfo `json:"info"`

		jsonErrorMsg
	}{
		Success:   err == nil,
		NewThread: newthread,
		Info:      pi,
	}
	if err != nil {
		ps.Code = code
		ps.Msg = err.Error()
	}

	e.Encode(ps)
}

func (j *JSONRenderer) WebCaptchaInclude(
	w http.ResponseWriter, r *http.Request) {

	// XXX do not make sense yet so do nothing
}
