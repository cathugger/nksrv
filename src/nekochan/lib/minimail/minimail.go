package minimail

import "io"

// some of types put in small package so that nntp won't need to pull in whole mail

type FullMsgID []byte // msgid with < and >
type CoreMsgID []byte // msgid without < and >
type FullMsgIDStr string
type CoreMsgIDStr string

type ArticleReader interface {
	io.Reader
	ReadByte() (byte, error)
	Discard(n int) (int, error)
}
