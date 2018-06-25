package testsrv

import (
	"fmt"
	"io"
	"time"

	. "nekochan/lib/logx"
	"nekochan/lib/nntp"
)

var _ nntp.NNTPProvider = (*TestSrv)(nil)

type groupState struct {
	group  string
	number uint64
	g      *group
}

type TestSrv struct {
	SupportNewNews     bool
	SupportOverByMsgID bool
	SupportHdr         bool
	SupportIHave       bool
	SupportPost        bool
	SupportStream      bool

	PostingPermit  bool
	PostingAccept  bool
	TransferPermit bool
	TransferAccept bool

	Log Logger
}

type (
	Responder    = nntp.Responder
	FullMsgID    = nntp.FullMsgID
	CoreMsgID    = nntp.CoreMsgID
	FullMsgIDStr = nntp.FullMsgIDStr
	CoreMsgIDStr = nntp.CoreMsgIDStr
	ConnState    = nntp.ConnState
)

func artnumInGroup(cs *ConnState, group string, num uint64) uint64 {
	if cg, ok := cs.CurrentGroup.(*groupState); ok && cg != nil && cg.group == group {
		return num
	} else {
		return 0
	}
}

func (p *TestSrv) SupportsNewNews() bool {
	return p.SupportNewNews
}

func (p *TestSrv) SupportsOverByMsgID() bool {
	return p.SupportOverByMsgID
}

func (p *TestSrv) SupportsHdr() bool {
	return p.SupportHdr
}

func (p *TestSrv) SupportsIHave() bool {
	return p.SupportIHave
}

func (p *TestSrv) SupportsPost() bool {
	return p.SupportPost
}

func (p *TestSrv) SupportsStream() bool {
	return p.SupportStream
}

func (p *TestSrv) GetArticleFullByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	a := s1.articles[sid]
	if a == nil {
		return false
	}
	w.ResArticleFollows(artnumInGroup(cs, a.group, a.number), a.msgid)
	dw := w.DotWriter()
	dw.Write(a.head)
	dw.Write(newline)
	dw.Write(a.body)
	dw.Close()
	return true
}

func (p *TestSrv) GetArticleHeadByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	a := s1.articles[sid]
	if a == nil {
		return false
	}
	w.ResHeadFollows(artnumInGroup(cs, a.group, a.number), a.msgid)
	dw := w.DotWriter()
	dw.Write(a.head)
	dw.Close()
	return true
}

func (p *TestSrv) GetArticleBodyByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	a := s1.articles[sid]
	if a == nil {
		return false
	}
	w.ResBodyFollows(artnumInGroup(cs, a.group, a.number), a.msgid)
	dw := w.DotWriter()
	dw.Write(a.body)
	dw.Close()
	return true
}

func (p *TestSrv) GetArticleStatByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	a := s1.articles[sid]
	if a == nil {
		return false
	}
	w.ResArticleFound(artnumInGroup(cs, a.group, a.number), a.msgid)
	return true
}

func getGroupState(cs *ConnState) *groupState {
	gs, _ := cs.CurrentGroup.(*groupState)
	return gs
}

func (p *TestSrv) GetArticleFullByNum(w Responder, cs *ConnState, num uint64) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[num]
	if a == nil {
		return false
	}
	w.ResArticleFollows(num, a.msgid)
	dw := w.DotWriter()
	dw.Write(a.head)
	dw.Write(newline)
	dw.Write(a.body)
	dw.Close()
	return true
}
func (p *TestSrv) GetArticleHeadByNum(w Responder, cs *ConnState, num uint64) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[num]
	if a == nil {
		return false
	}
	w.ResHeadFollows(num, a1msgid)
	dw := w.DotWriter()
	dw.Write(a.head)
	dw.Close()
	return true
}
func (p *TestSrv) GetArticleBodyByNum(w Responder, cs *ConnState, num uint64) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[num]
	if a == nil {
		return false
	}
	w.ResBodyFollows(num, a.msgid)
	dw := w.DotWriter()
	dw.Write(a.body)
	dw.Close()
	return true
}
func (p *TestSrv) GetArticleStatByNum(w Responder, cs *ConnState, num uint64) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[num]
	if a == nil {
		return false
	}
	w.ResArticleFound(num, a.msgid)
	return true
}

func (p *TestSrv) GetArticleFullByCurr(w Responder, cs *ConnState) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[gs.number]
	if a == nil {
		return false
	}
	w.ResArticleFollows(a.number, a.msgid)
	dw := w.DotWriter()
	dw.Write(a.head)
	dw.Write(newline)
	dw.Write(a.body)
	dw.Close()
	return true
}
func (p *TestSrv) GetArticleHeadByCurr(w Responder, cs *ConnState) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[gs.number]
	if a == nil {
		return false
	}
	w.ResHeadFollows(a.number, a.msgid)
	dw := w.DotWriter()
	dw.Write(a.head)
	dw.Close()
	return true
}
func (p *TestSrv) GetArticleBodyByCurr(w Responder, cs *ConnState) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[gs.number]
	if a == nil {
		return false
	}
	w.ResBodyFollows(a.number, a.msgid)
	dw := w.DotWriter()
	dw.Write(a.body)
	dw.Close()
	return true
}
func (p *TestSrv) GetArticleStatByCurr(w Responder, cs *ConnState) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[gs.number]
	if a == nil {
		return false
	}
	w.ResArticleFound(a.number, a.msgid)
	return true
}

func (p *TestSrv) SelectGroup(w Responder, cs *ConnState, group []byte) bool {
	sgroup := unsafeBytesToStr(group)

	g := s1.groups[sgroup]
	if g == nil {
		return false
	}

	gs := getGroupState(cs)
	if gs == nil {
		gs = &groupState{}
		cs.CurrentGroup = gs
	}

	gs.group = sgroup
	gs.g = g

	if len(g.articlesSort) != 0 {
		gs.number = g.articlesSort[0]

		lo, hi := g.articlesSort[0], g.articlesSort[len(g.articlesSort)-1]
		w.ResGroupSuccessfullySelected(hi-lo+1, lo, hi, sgroup)
	} else {
		gs.number = 0

		w.ResGroupSuccessfullySelected(0, 0, 0, gs.group)
	}

	return true
}
func (p *TestSrv) SelectAndListGroup(w Responder, cs *ConnState, group []byte, rmin, rmax int64) bool {
	gs := getGroupState(cs)
	if gs == nil {
		if len(group) == 0 {
			w.ResNoNewsgroupSelected()
			return true
		}
		gs = &groupState{}
		cs.CurrentGroup = gs
	}

	var sgroup string
	if len(group) != 0 {
		sgroup = unsafeBytesToStr(group)
	} else {
		sgroup = gs.group
	}

	g := s1.groups[sgroup]
	if g == nil {
		return false
	}

	gs.group = g.name
	gs.g = g

	if len(g.articlesSort) != 0 {
		gs.number = g.articlesSort[0]

		lo, hi := g.articlesSort[0], g.articlesSort[len(g.articlesSort)-1]
		w.ResArticleNumbersFollow(hi-lo+1, lo, hi, gs.group)
	} else {
		gs.number = 0

		w.ResArticleNumbersFollow(0, 0, 0, gs.group)
	}

	dw := w.DotWriter()
	for _, n := range g.articlesSort {
		if n >= uint64(rmin) && (rmax < 0 || n <= uint64(rmax)) {
			fmt.Fprintf(dw, "%d\n", n)
		}
	}
	dw.Close()

	return true
}
func (p *TestSrv) SelectNextArticle(w Responder, cs *ConnState) {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return
	}
	x := gs.number
	if x == 0 {
		w.ResCurrentArticleNumberIsInvalid()
		return
	}
	for _, n := range gs.g.articlesSort {
		if n > x {
			gs.number = n
			a := gs.g.articles[n]
			w.ResArticleFound(a.number, a.msgid)
			return
		}
	}
	w.ResNoNextArticleInThisGroup()
}
func (p *TestSrv) SelectPrevArticle(w Responder, cs *ConnState) {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return
	}
	x := gs.number
	if x == 0 {
		w.ResCurrentArticleNumberIsInvalid()
		return
	}
	for i := len(gs.g.articlesSort) - 1; i >= 0; i-- {
		n := gs.g.articlesSort[i]
		if n < x {
			gs.number = n
			a := gs.g.articles[n]
			w.ResArticleFound(a.number, a.msgid)
			return
		}
	}
	w.ResNoPrevArticleInThisGroup()
}

func (p *TestSrv) ListNewGroups(w io.Writer, qt time.Time) {
	for _, gn := range s1.groupsSort {
		g := s1.groups[gn]
		if !qt.After(g.created) {
			lo, hi := g.articlesSort[0], g.articlesSort[len(g.articlesSort)-1]
			fmt.Fprintf(w, "%s %d %d %c\n", gn, hi, lo, g.status)
		}
	}
}

func (p *TestSrv) ListNewNews(w io.Writer, wildmat []byte, qt time.Time) {
	var chk func(string) bool
	if len(wildmat) == 0 {
		chk = func(g string) bool { return true }
	} else {
		if nntp.ValidGroupSlice(wildmat) {
			sw := unsafeBytesToStr(wildmat)
			chk = func(g string) bool { return g == sw }
		} else {
			wm := nntp.CompileWildmat(wildmat)
			chk = func(g string) bool { return wm.CheckString(g) }
		}
	}

	for id, a := range s1.articles {
		if !qt.After(a.posted) && chk(a.group) {
			fmt.Fprintf(w, "<%s>\n", id)
		}
	}
}

func (p *TestSrv) ListActiveGroups(w io.Writer, wildmat []byte) {
	var chk func(string) bool
	if len(wildmat) == 0 {
		chk = func(g string) bool { return true }
	} else {
		if nntp.ValidGroupSlice(wildmat) {
			sw := unsafeBytesToStr(wildmat)
			chk = func(g string) bool { return g == sw }
		} else {
			wm := nntp.CompileWildmat(wildmat)
			chk = func(g string) bool { return wm.CheckString(g) }
		}
	}

	for _, gn := range s1.groupsSort {
		if chk(gn) {
			g := s1.groups[gn]
			lo, hi := g.articlesSort[0], g.articlesSort[len(g.articlesSort)-1]
			fmt.Fprintf(w, "%s %d %d %c\n", gn, hi, lo, g.status)
		}
	}
}

func (p *TestSrv) ListNewsgroups(w io.Writer, wildmat []byte) {
	var chk func(string) bool
	if len(wildmat) == 0 {
		chk = func(g string) bool { return true }
	} else {
		if nntp.ValidGroupSlice(wildmat) {
			sw := unsafeBytesToStr(wildmat)
			chk = func(g string) bool { return g == sw }
		} else {
			wm := nntp.CompileWildmat(wildmat)
			chk = func(g string) bool { return wm.CheckString(g) }
		}
	}

	for _, gn := range s1.groupsSort {
		if chk(gn) {
			g := s1.groups[gn]
			fmt.Fprintf(w, "%s\t%s\n", gn, g.info)
		}
	}
}

func printOver(w io.Writer, num uint64, a *article) {
	fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", num,
		a.over.subject, a.over.from, a.over.date, a.over.msgid,
		a.over.references, a.over.bytes, a.over.lines, a.over.xref)
}

// + ok: 224{ResOverviewInformationFollows}
// fail:
//   <ByMsgID>      430{ResNoArticleWithThatMsgID[false]}
//   <OverByRange>  412{ResNoNewsgroupSelected} 423{ResNoArticlesInThatRange[false]}
//   <XOverByRange> 412{ResNoNewsgroupSelected} 420{ResXNoArticles[false]}
//   <ByCurr>       412{ResNoNewsgroupSelected} 420{ResCurrentArticleNumberIsInvalid[false]}
func (p *TestSrv) GetOverByMsgID(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	a := s1.articles[sid]
	if a == nil {
		return false
	}
	/*
		The first 8 fields MUST be the following, in order:
			"0" or article number (see below)
			Subject header content
			From header content
			Date header content
			Message-ID header content
			References header content
			:bytes metadata item
			:lines metadata item
	*/
	w.ResOverviewInformationFollows()
	dw := w.DotWriter()
	printOver(dw, artnumInGroup(cs, a.group, a.number), a)
	dw.Close()
	return true
}
func (p *TestSrv) GetOverByRange(w Responder, cs *ConnState, rmin, rmax int64) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	var ww io.WriteCloser = nil
	for _, an := range gs.g.articlesSort {
		if an >= uint64(rmin) && (rmax < 0 || an <= uint64(rmax)) {
			if ww == nil {
				w.ResOverviewInformationFollows()
				ww = w.DotWriter()
			}
			a := gs.g.articles[an]
			printOver(ww, a.number, a)
		}
	}
	if ww != nil {
		ww.Close()
		return true
	} else {
		return false
	}
}
func (p *TestSrv) GetXOverByRange(w Responder, cs *ConnState, rmin, rmax int64) bool {
	return p.GetOverByRange(w, cs, rmin, rmax)
}
func (p *TestSrv) GetOverByCurr(w Responder, cs *ConnState) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[gs.number]
	if a == nil {
		return false
	}
	w.ResOverviewInformationFollows()
	dw := w.DotWriter()
	printOver(dw, a.number, a)
	dw.Close()
	return true
}

func (p *TestSrv) commonGetHdrByMsgID(w Responder, cs *ConnState, hdr []byte, msgid CoreMsgID, rfc bool) bool {
	sid := unsafeCoreMsgIDToStr(msgid)
	a := s1.articles[sid]
	if a == nil {
		return false
	}
	h, supported := a.over.GetByHdr(hdr)
	if !supported {
		w.PrintfLine("503 %q header unsupported", hdr)
		return true
	}
	if rfc {
		w.ResHdrFollow()
		dw := w.DotWriter()
		fmt.Fprintf(dw, "%d %s\n", artnumInGroup(cs, a.group, a.number), h)
		dw.Close()
	} else {
		w.ResXHdrFollow()
		dw := w.DotWriter()
		fmt.Fprintf(dw, "<%s> %s\n", msgid, h)
		dw.Close()
	}
	return true
}
func (p *TestSrv) commonGetHdrByRange(w Responder, cs *ConnState, hdr []byte, rmin, rmax int64, rfc bool) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	var ww io.WriteCloser = nil
	for _, an := range gs.g.articlesSort {
		if an >= uint64(rmin) && (rmax < 0 || an <= uint64(rmax)) {
			a := gs.g.articles[an]
			h, supported := a.over.GetByHdr(hdr)
			if !supported {
				w.PrintfLine("503 %q header unsupported", hdr)
				return true
			}
			if ww == nil {
				if rfc {
					w.ResHdrFollow()
				} else {
					w.ResXHdrFollow()
				}
				ww = w.DotWriter()
			}
			fmt.Fprintf(ww, "%d %s\n", an, h)
		}
	}
	if ww != nil {
		ww.Close()
		return true
	} else {
		return false
	}
}
func (p *TestSrv) commonGetHdrByCurr(w Responder, cs *ConnState, hdr []byte, rfc bool) bool {
	gs := getGroupState(cs)
	if gs == nil {
		w.ResNoNewsgroupSelected()
		return true
	}
	a := gs.g.articles[gs.number]
	if a == nil {
		return false
	}
	h, supported := a.over.GetByHdr(hdr)
	if !supported {
		w.PrintfLine("503 %q header unsupported", hdr)
		return true
	}
	if rfc {
		w.ResHdrFollow()
	} else {
		w.ResXHdrFollow()
	}
	dw := w.DotWriter()
	fmt.Fprintf(dw, "%d %s\n", a.number, h)
	dw.Close()
	return true
}
func (p *TestSrv) GetHdrByMsgID(w Responder, cs *ConnState, hdr []byte, msgid CoreMsgID) bool {
	return p.commonGetHdrByMsgID(w, cs, hdr, msgid, true)
}
func (p *TestSrv) GetHdrByRange(w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool {
	return p.commonGetHdrByRange(w, cs, hdr, rmin, rmax, true)
}
func (p *TestSrv) GetHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool {
	return p.commonGetHdrByCurr(w, cs, hdr, true)
}
func (p *TestSrv) GetXHdrByMsgID(w Responder, hdr []byte, msgid CoreMsgID) bool {
	return p.commonGetHdrByMsgID(w, nil, hdr, msgid, false)
}
func (p *TestSrv) GetXHdrByRange(w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool {
	return p.commonGetHdrByRange(w, cs, hdr, rmin, rmax, false)
}
func (p *TestSrv) GetXHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool {
	return p.commonGetHdrByCurr(w, cs, hdr, false)
}
