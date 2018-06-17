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
	GetArticleFullByMsgID(w Responder, msgid []byte) bool
	GetArticleHeadByMsgID(w Responder, msgid []byte) bool
	GetArticleBodyByMsgID(w Responder, msgid []byte) bool
	GetArticleStatByMsgID(w Responder, msgid []byte) bool
	GetArticleFullByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleHeadByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleBodyByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleStatByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleFullByCurr(w Responder, cs *ConnState) bool
	GetArticleHeadByCurr(w Responder, cs *ConnState) bool
	GetArticleBodyByCurr(w Responder, cs *ConnState) bool
	GetArticleStatByCurr(w Responder, cs *ConnState) bool

	SetGroup(w Responder, cs *ConnState, group []byte) bool
	SelectNext(w Responder, cs *ConnState)
	SelectPrev(w Responder, cs *ConnState)
}
