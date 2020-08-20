package webib0

// unused, TODO rethink/remove

import (
	"io"
)

type BodyType int

const (
	StringBody BodyType = iota
	ExternalBody
	Base64Body
)

type (
	PostInfo struct {
		Root  MsgRoot
		Files map[string]PostFile
	}

	PostFile interface {
		io.Writer
		Delete()
	}

	MsgRoot struct {
		Head map[string][]HeaderVal
		Body interface{} // can be single or composite
	}

	PlainBody struct {
		Type  BodyType
		Value []byte
	}

	MultipartBody []Part

	Part struct {
		Head []PartHeader
		Body interface{} // can be single or composite
	}

	PartHeader struct {
		Key string
		Val HeaderVal
	}

	HeaderVal []byte
)

func (pi *PostInfo) DeleteFiles() {
	for k := range pi.Files {
		pi.Files[k].Delete()
		delete(pi.Files, k)
	}
}

// for cases when we intend to store some stuff in memory,
// but only limited amount of it
type PostContext interface {
	MakeFile() (PostFile, error)
	Release()
}

type PostProvider interface {
	NewContext() PostContext
	Submit(p *PostInfo) error
}
