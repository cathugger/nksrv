package apiproxy

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ib0 "centpd/lib/webib0"
)

type APIProxy struct {
	c      *http.Client
	apiURL string
}

var _ ib0.IBProvider = APIProxy{}

func NewAPIProxy(apiURL string) APIProxy {
	return APIProxy{
		c: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiURL: apiURL,
	}
}

func (p APIProxy) call(path string, ret interface{}) (error, int) {
	r, err := p.c.Get(p.apiURL + path)
	if err != nil {
		return err, 502
	}
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	err = dec.Decode(ret)
	if err != nil {
		return err, 502
	}

	return nil, 0
}

func (p APIProxy) IBGetBoardList(blp *ib0.IBBoardList) (error, int) {
	return p.call("/boards/", blp)
}

func (p APIProxy) IBGetThreadListPage(tlp *ib0.IBThreadListPage, b string, pn uint32) (error, int) {
	return p.call("/boards/"+b+"/pages/"+strconv.FormatUint(uint64(pn), 10), tlp)
}

func (p APIProxy) IBGetOverboardPage(op *ib0.IBOverboardPage, pn uint32) (error, int) {
	return p.call("/overboard/pages/"+strconv.FormatUint(uint64(pn), 10), op)
}

func (p APIProxy) IBGetThread(tp *ib0.IBThreadPage, b string, t string) (error, int) {
	return p.call("/boards/"+b+"/threads/"+t, tp)
}

func (p APIProxy) IBGetThreadCatalog(tcp *ib0.IBThreadCatalog, b string) (error, int) {
	return p.call("/boards/"+b+"/catalog", tcp)
}
