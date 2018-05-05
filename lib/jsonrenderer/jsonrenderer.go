package jsonrenderer

import (
	//. "../logx"
	"../renderer"
	ib0 "../webib0"
	"encoding/json"
	"net/http"
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
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type jsonError struct {
	Err jsonErrorMsg `json:"error"`
}

func (j *JSONRenderer) prepareEncoder(w http.ResponseWriter) *json.Encoder {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	e.SetIndent("", j.indent)
	return e
}

func returnError(w http.ResponseWriter, e *json.Encoder, err error, code int) {
	w.WriteHeader(code)
	jerr := jsonError{Err: jsonErrorMsg{Code: code, Msg: err.Error()}}
	e.Encode(&jerr)
}

func (j *JSONRenderer) ServeBoardList(w http.ResponseWriter, r *http.Request) {
	e := j.prepareEncoder(w)
	var list ib0.IBBoardList
	err, code := j.p.IBGetBoardList(&list)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&list)
}

func (j *JSONRenderer) ServeThreadListPage(w http.ResponseWriter, r *http.Request, board string, page uint32) {
	e := j.prepareEncoder(w)
	var pag ib0.IBThreadListPage
	err, code := j.p.IBGetThreadListPage(&pag, board, page)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&pag)
}

func (j *JSONRenderer) ServeThreadCatalog(w http.ResponseWriter, r *http.Request, board string) {
	e := j.prepareEncoder(w)
	var pag ib0.IBThreadCatalog
	err, code := j.p.IBGetThreadCatalog(&pag, board)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&pag)
}

func (j *JSONRenderer) ServeThread(w http.ResponseWriter, r *http.Request, board, thread string) {
	e := j.prepareEncoder(w)
	var pag ib0.IBThreadPage
	err, code := j.p.IBGetThread(&pag, board, thread)
	if err != nil {
		returnError(w, e, err, code)
		return
	}
	e.Encode(&pag)
}
