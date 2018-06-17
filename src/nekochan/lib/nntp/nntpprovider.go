package nntp

import (
	tp "net/textproto"
)

// sugar because im lazy
type Responder struct {
	*tp.Writer
}

type NNTPProvider interface {
	// ARTICLE, HEAD, BODY, STAT x 3 forms for each
	// ok: ARTICLE - 220, HEAD - 221, BODY - 222, STAT - 223
	// fail:
	//   1st_form: 430 (not found by msgid)
	//   2nd_form: 412 (no group selected), 423 (not found by num)
	//   3rd_form: 412 (no group selected), 420 (not found by curr)
	GetArticleFullByMsgID(w Responder, msgid []byte)
	GetArticleHeadByMsgID(w Responder, msgid []byte)
	GetArticleBodyByMsgID(w Responder, msgid []byte)
	GetArticleStatByMsgID(w Responder, msgid []byte)
	GetArticleFullByNum(w Responder, cs *ConnState, num uint64)
	GetArticleHeadByNum(w Responder, cs *ConnState, num uint64)
	GetArticleBodyByNum(w Responder, cs *ConnState, num uint64)
	GetArticleStatByNum(w Responder, cs *ConnState, num uint64)
	GetArticleFullByCurr(w Responder, cs *ConnState)
	GetArticleHeadByCurr(w Responder, cs *ConnState)
	GetArticleBodyByCurr(w Responder, cs *ConnState)
	GetArticleStatByCurr(w Responder, cs *ConnState)
}
