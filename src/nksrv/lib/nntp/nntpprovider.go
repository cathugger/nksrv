package nntp

import (
	"io"
	"time"

	mail "nksrv/lib/utils/minimail"
)

type TFullMsgID = mail.TFullMsgID // msgid with < and >
type TCoreMsgID = mail.TCoreMsgID // msgid without < and >
type TFullMsgIDStr = mail.TFullMsgIDStr
type TCoreMsgIDStr = mail.TCoreMsgIDStr

type ArticleReader = mail.ArticleReader

type ReaderOpener interface {
	OpenReader() ArticleReader
}

type AbstractResponder interface {
	OpenDotWriter() (io.WriteCloser, error)
	GetResponder() Responder
	Abort()
}

type NNTPProvider interface {
	SupportsNewNews() bool
	SupportsOverByMsgID() bool
	SupportsHdr() bool
	SupportsIHave() bool
	SupportsPost() bool
	SupportsStream() bool
	SupportsXListen() bool

	// + ARTICLE, HEAD, BODY, STAT x 3 forms for each
	// ok:
	//   ARTICLE Full 220{ResArticleFollows}
	//   HEAD    Head 221{ResHeadFollows}
	//   BODY    Body 222{ResBodyFollows}
	//   STAT    Stat 223{ResArticleFound}
	// fail:
	//   <ByMsgID> 430{ResNoArticleWithThatMsgID[false]}
	//   <ByNum>   412{ResNoNewsgroupSelected} 423{ResNoArticleWithThatNum[false]}
	//   <ByCurr>  412{ResNoNewsgroupSelected} 420{ResCurrentArticleNumberIsInvalid[false]}
	GetArticleFullByMsgID(w Responder, cs *ConnState, msgid TCoreMsgID) bool
	GetArticleHeadByMsgID(w Responder, cs *ConnState, msgid TCoreMsgID) bool
	GetArticleBodyByMsgID(w Responder, cs *ConnState, msgid TCoreMsgID) bool
	GetArticleStatByMsgID(w Responder, cs *ConnState, msgid TCoreMsgID) bool
	GetArticleFullByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleHeadByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleBodyByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleStatByNum(w Responder, cs *ConnState, num uint64) bool
	GetArticleFullByCurr(w Responder, cs *ConnState) bool
	GetArticleHeadByCurr(w Responder, cs *ConnState) bool
	GetArticleBodyByCurr(w Responder, cs *ConnState) bool
	GetArticleStatByCurr(w Responder, cs *ConnState) bool

	// + ok: 211{ResGroupSuccessfullySelected} fail: 411{ResNoSuchNewsgroup[false]}
	SelectGroup(w Responder, cs *ConnState, group []byte) bool
	// + ok: 211{ResArticleNumbersFollow} fail: 411{ResNoSuchNewsgroup[false]} 412{ResNoNewsgroupSelected}
	SelectAndListGroup(w Responder, cs *ConnState, group []byte, rmin, rmax int64) bool
	// + ok: 223{ResArticleFound} fail: 412{ResNoNewsgroupSelected} 420{ResCurrentArticleNumberIsInvalid}
	// - fail: 421{ResNoNextArticleInThisGroup}
	SelectNextArticle(w Responder, cs *ConnState)
	// - fail: 422{ResNoPrevArticleInThisGroup}
	SelectPrevArticle(w Responder, cs *ConnState)

	// + 230{ResListOfNewArticlesFollows} ret: list of FullMsgID
	ListNewNews(aw AbstractResponder, wildmat []byte, qt time.Time) // SupportsNewNews()
	// + 231{ResListOfNewNewsgroupsFollows} ret: list of {name hiwm lowm status}
	ListNewGroups(aw AbstractResponder, qt time.Time)
	// + 215{ResListFollows}
	ListActiveGroups(aw AbstractResponder, wildmat []byte)
	ListNewsgroups(aw AbstractResponder, wildmat []byte)

	// + ok: 224{ResOverviewInformationFollows}
	// fail:
	//   <ByMsgID>      430{ResNoArticleWithThatMsgID[false]}
	//   <OverByRange>  412{ResNoNewsgroupSelected} 423{ResNoArticlesInThatRange[false]}
	//   <XOverByRange> 412{ResNoNewsgroupSelected} 420{ResXNoArticles[false]}
	//   <ByCurr>       412{ResNoNewsgroupSelected} 420{ResCurrentArticleNumberIsInvalid[false]}
	GetOverByMsgID(w Responder, cs *ConnState, msgid TCoreMsgID) bool // SupportsOverByMsgID()
	GetOverByRange(w Responder, cs *ConnState, rmin, rmax int64) bool
	GetXOverByRange(w Responder, cs *ConnState, rmin, rmax int64) bool
	GetOverByCurr(w Responder, cs *ConnState) bool

	// + SupportsHdr()
	//   <HdrByMsgID>  ok: 225{ResHdrFollow}  fail: 430{ResNoArticleWithThatMsgID[false]}
	//   <HdrByRange>  ok: 225{ResHdrFollow}  fail: 412{ResNoNewsgroupSelected} 423{ResNoArticlesInThatRange[false]}
	//   <HdrByCurr>   ok: 225{ResHdrFollow}  fail: 412{ResNoNewsgroupSelected} 420{ResCurrentArticleNumberIsInvalid[false]}
	//   <XHdrByMsgID> ok: 221{ResXHdrFollow} fail: 430{ResNoArticleWithThatMsgID[false]}
	//   <XHdrByRange> ok: 221{ResXHdrFollow} fail: 412{ResNoNewsgroupSelected} 420{ResXNoArticles[false]}
	//   <XHdrByCurr>  ok: 221{ResXHdrFollow} fail: 412{ResNoNewsgroupSelected} 420{ResCurrentArticleNumberIsInvalid[false]}
	GetHdrByMsgID(w Responder, cs *ConnState, hdr []byte, msgid TCoreMsgID) bool
	GetHdrByRange(w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool
	GetHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool
	GetXHdrByMsgID(w Responder, hdr []byte, msgid TCoreMsgID) bool
	GetXHdrByRange(w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool
	GetXHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool

	// ! implementers MUST drain readers or bad things will happen
	// + iok: 340{ResSendArticleToBePosted} ifail: 440{ResPostingNotPermitted[false]}
	// cok: 240{ResPostingAccepted} cfail: 441{ResPostingFailed}
	HandlePost(w Responder, cs *ConnState, ro ReaderOpener) bool // SupportsPost()
	// + iok: 335{ResSendArticleToBeTransferred} ifail: 435{ResTransferNotWanted[false]} 436{ResTransferFailed}
	// cok: 235{ResTransferSuccess} cfail: 436{ResTransferFailed} 437{ResTransferRejected}
	HandleIHave(w Responder, cs *ConnState, ro ReaderOpener, msgid TCoreMsgID) bool // SupportsIHave()
	// + ok: 238{ResArticleWanted} fail: 431{ResArticleWantLater} 438{ResArticleNotWanted[false]}
	HandleCheck(w Responder, cs *ConnState, msgid TCoreMsgID) bool // SupportsStream()
	// + ok: 239{ResArticleTransferedOK} 439{ResArticleRejected[false]}
	HandleTakeThis(w Responder, cs *ConnState, r ArticleReader, msgid TCoreMsgID) bool // SupportsStream()
}
