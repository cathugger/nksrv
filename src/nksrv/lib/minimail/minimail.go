package minimail

import (
	"io"

	au "nksrv/lib/asciiutils"
)

// some of types put in small package so that nntp won't need to pull in whole mail

type TFullMsgID []byte // msgid including < and >
type TCoreMsgID []byte // msgid excluding < and >
type TFullMsgIDStr string
type TCoreMsgIDStr string

type ArticleReader interface {
	io.Reader
	ReadByte() (byte, error)
	Discard(n int) (int, error)
	InvalidNL() bool
}

func CutMessageIDStr(id TFullMsgIDStr) TCoreMsgIDStr {
	return TCoreMsgIDStr(id[1 : len(id)-1])
}

func ValidMessageIDStr(id TFullMsgIDStr) bool {
	return len(id) >= 3 &&
		id[0] == '<' && id[len(id)-1] == '>' && len(id) <= 250 &&
		au.IsPrintableASCIIStr(string(CutMessageIDStr(id)), '>')
}
