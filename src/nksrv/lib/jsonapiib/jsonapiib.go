package jsonapiib

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"nksrv/lib/webib0"
)

type JSONAPIIB struct {
	c http.Client
	t http.Transport
	u string
}

var _ webib0.IBProvider = (*JSONAPIIB)(nil)

type jsonErrorMsg struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

type jsonError struct {
	Err jsonErrorMsg `json:"error"`
}

func NewJSONAPIIB(u string) (a *JSONAPIIB) {

	// cut off trailing /
	if len(u) != 0 && u[len(u)-1] == '/' {
		u = u[:len(u)-1]
	}

	// some default settings idk if overriding these makes sense
	a = &JSONAPIIB{
		t: http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		},
		u: u,
	}

	a.c.Transport = &a.t

	return
}

func isJSONType(t string) bool {
	return t == "application/json"
}

func fetchInto(
	c *http.Client, u string, d interface{}) (error, int) {

	resp, err := c.Get(u)
	if err != nil {
		return fmt.Errorf("API request error: %v", err), 502
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !isJSONType(ct) {
		return fmt.Errorf("API returned bad Content-Type %q", ct), 502
	}

	jd := json.NewDecoder(resp.Body)

	resc := resp.StatusCode
	if resc != 200 {
		var je jsonError
		err = jd.Decode(&je)
		if err != nil {
			return fmt.Errorf(
				"error parsing json error on code %d: %v", resc, err), 502
		}
		if je.Err.Msg == "" {
			return fmt.Errorf("empty err msg on code %d", resc), 502
		}
		return errors.New(je.Err.Msg), je.Err.Code
	}

	err = jd.Decode(d)
	if err != nil {
		return fmt.Errorf("error parsing json response: %v", err), 502
	}

	return nil, resc
}

func (a *JSONAPIIB) IBGetBoardList(page *webib0.IBBoardList) (error, int) {
	return fetchInto(&a.c, a.u+"/boards/", page)
}

func (a *JSONAPIIB) IBGetThreadListPage(
	page *webib0.IBThreadListPage, board string, num uint32) (error, int) {

	return fetchInto(
		&a.c,
		a.u+"/boards/"+url.PathEscape(board)+"/"+strconv.FormatUint(uint64(num), 10),
		page)
}

func (a *JSONAPIIB) IBGetOverboardPage(
	page *webib0.IBOverboardPage, num uint32) (error, int) {

	return fetchInto(
		&a.c,
		a.u+"/overboard/"+strconv.FormatUint(uint64(num), 10),
		page)
}

func (a *JSONAPIIB) IBGetThreadCatalog(
	page *webib0.IBThreadCatalog, board string) (error, int) {

	return fetchInto(&a.c, a.u+"/boards/"+url.PathEscape(board)+"/catalog", page)
}

func (a *JSONAPIIB) IBGetOverboardCatalog(
	page *webib0.IBOverboardCatalog) (error, int) {

	return fetchInto(&a.c, a.u+"/overboard/catalog", page)
}

func (a *JSONAPIIB) IBGetThread(
	page *webib0.IBThreadPage, board string, threadid string) (error, int) {

	return fetchInto(
		&a.c,
		a.u+"/boards/"+url.PathEscape(board)+"/threads/"+threadid,
		page)
}
