package testsrv

import (
	. "nekochan/lib/logx"
	"nekochan/lib/nntp"
)

func validMsgID(s FullMsgIDStr) bool {
	return nntp.ValidMessageID(unsafeStrToBytes(string(s)))
}

func reservedMsgID(s FullMsgIDStr) bool {
	return nntp.ReservedMessageID(unsafeStrToBytes(string(s)))
}

func cutMsgID(s FullMsgIDStr) CoreMsgIDStr {
	return CoreMsgIDStr(unsafeBytesToStr(nntp.CutMessageID(unsafeStrToBytes(string(s)))))
}

func unsafeCoreMsgIDToStr(b CoreMsgID) CoreMsgIDStr {
	return CoreMsgIDStr(unsafeBytesToStr(b))
}

func getHdrMsgID(h nntp.Headers) FullMsgIDStr {
	return FullMsgIDStr(nntp.TrimWSStr(string(h.GetFirst("Message-ID"))))
}

// ! implementers MUST drain readers or bad things will happen
// + iok: 340{ResSendArticleToBePosted} ifail: 440{ResPostingNotPermitted[false]}
// cok: 240{ResPostingAccepted} cfail: 441{ResPostingFailed}
func (p *TestSrv) HandlePost(w Responder, cs *ConnState, ro nntp.ReaderOpener) bool {
	if !p.PostingPermit {
		return false
	}
	w.ResSendArticleToBePosted()
	r := ro.OpenReader()
	h, e := nntp.ReadHeaders(r, 2<<20)
	p.Log.LogPrintf(DEBUG, "finished reading headers")
	if e != nil {
		p.Log.LogPrintf(WARN, "header parsing error: %v", e)
	}
	mid := getHdrMsgID(h.H)
	if len(mid) != 0 {
		if !validMsgID(mid) {
			p.Log.LogPrintf(DEBUG, "invalid msg id %q", mid)
		} else if reservedMsgID(mid) {
			p.Log.LogPrintf(DEBUG, "reserved msg id %q", mid)
		}
	}
	if p.Log.Level() <= DEBUG {
		for x, v := range h.H {
			p.Log.LogPrintf(DEBUG, "header[%q]: %v", x, v)
		}
	}
	if !p.PostingAccept || e != nil || (len(mid) != 0 && (!validMsgID(mid) || reservedMsgID(mid) || s1.articles[cutMsgID(mid)] != nil)) {
		w.ResPostingFailed()
		h.Close()
		r.Discard(-1)
		return true
	}
	h.B.Discard(-1)
	h.Close()
	r.Discard(-1) // ensure
	w.ResPostingAccepted()
	return true
}

// + iok: 335{ResSendArticleToBeTransferred} ifail: 435{ResTransferNotWanted[false]} 436{ResTransferFailed}
// cok: 235{ResTransferSuccess} cfail: 436{ResTransferFailed} 437{ResTransferRejected}
func (p *TestSrv) HandleIHave(w Responder, cs *ConnState, ro nntp.ReaderOpener, msgid CoreMsgID) bool {
	if !p.TransferPermit {
		w.ResTransferFailed()
		return true
	}
	mstr := unsafeCoreMsgIDToStr(msgid)
	if s1.articles[mstr] != nil {
		return false
	}
	w.ResSendArticleToBeTransferred()
	r := ro.OpenReader()
	h, e := nntp.ReadHeaders(r, 2<<20)
	mid := getHdrMsgID(h.H)
	if !p.TransferAccept || e != nil || !validMsgID(mid) || cutMsgID(mid) != mstr {
		w.ResTransferRejected()
		h.Close()
		r.Discard(-1)
		return true
	}
	h.B.Discard(-1)
	h.Close()
	r.Discard(-1) // ensure
	w.ResTransferSuccess()
	return true
}

// + ok: 238{ResPleaseSend} fail: 431{ResCantAccept} 438{ResArticleNotWanted[false]}
func (p *TestSrv) HandleCheck(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	if !p.TransferPermit {
		w.ResCantAccept(msgid)
		return true
	}
	if s1.articles[unsafeCoreMsgIDToStr(msgid)] != nil {
		return false
	}
	w.ResPleaseSend(msgid)
	return true
}

// + ok: 239{ResArticleTransferedOK} 439{ResArticleRejected[false]}
func (p *TestSrv) HandleTakeThis(w Responder, cs *ConnState, r nntp.ArticleReader, msgid CoreMsgID) bool {
	h, e := nntp.ReadHeaders(r, 2<<20)
	mid := getHdrMsgID(h.H)
	if !p.TransferAccept || e != nil || !validMsgID(mid) || cutMsgID(mid) != unsafeCoreMsgIDToStr(msgid) {
		w.ResArticleRejected(msgid)
		h.Close()
		r.Discard(-1)
		return true
	}
	h.B.Discard(-1)
	h.Close()
	r.Discard(-1) // ensure
	w.ResArticleTransferedOK(msgid)
	return true
}
