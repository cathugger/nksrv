package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"

	"nksrv/lib/mail/form"
)

type fo struct{}

func (fo) OpenFile() (*os.File, error) {
	return ioutil.TempFile("", "formdata-")
}

func testf1(w http.ResponseWriter, r *http.Request) {
	ct, param, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil {
		http.Error(w, fmt.Sprintf("failed to parse content type: %v", e), 400)
		return
	}
	if ct != "multipart/form-data" || param["boundary"] == "" {
		http.Error(w, "bad Content-Type", 400)
		return
	}
	f, e := form.ParseForm(
		r.Body, param["boundary"],
		form.FieldsCheckFunc([]string{"aaa"}),
		form.FieldsCheckFunc([]string{"bbb"}), fo{})
	if e != nil {
		http.Error(w, fmt.Sprintf("error parsing form: %v", e), 400)
		return
	}
	fmt.Fprintf(w, "lol finished looking thru it\n")
	f.RemoveAll()
}

func main() {
	sm := http.NewServeMux()
	sm.Handle("/test1", http.HandlerFunc(testf1))
	s := &http.Server{
		Addr:           ":4321",
		Handler:        sm,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
