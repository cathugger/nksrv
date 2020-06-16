package psqlib

var _ nntp.NNTPProvider = (*PSQLIB)(nil)

func (*PSQLIB) SupportsNewNews() bool     { return true }
func (*PSQLIB) SupportsOverByMsgID() bool { return true }
func (*PSQLIB) SupportsHdr() bool         { return true }

func (p *PSQLIB) SupportsIHave() bool  { return true }
func (p *PSQLIB) SupportsPost() bool   { return true }
func (p *PSQLIB) SupportsStream() bool { return true }

func (p *PSQLIB) SupportsXListen() bool { return false }

// ARTICLE/HEAD/BODY/STAT by MsgID
func (sp *PSQLIB) GetArticleFullByMsgID(
	w Responder, cs *ConnState, msgid TCoreMsgID) bool {

	nc := &pireadnntp.FullNNTPCopyer{W: w}
	return pireadnntp.GetArticleCommonByMsgID(&sp.PSQLIB, nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleHeadByMsgID(
	w Responder, cs *ConnState, msgid TCoreMsgID) bool {

	nc := &pireadnntp.HeadNNTPCopyer{W: w}
	return pireadnntp.GetArticleCommonByMsgID(&sp.PSQLIB, nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleBodyByMsgID(
	w Responder, cs *ConnState, msgid TCoreMsgID) bool {

	nc := &pireadnntp.BodyNNTPCopyer{W: w}
	return pireadnntp.GetArticleCommonByMsgID(&sp.PSQLIB, nc, w, cs, msgid)
}
func (sp *PSQLIB) GetArticleStatByMsgID(
	w Responder, cs *ConnState, msgid TCoreMsgID) bool {

	nc := &pireadnntp.StatNNTPCopyer{W: w}
	return pireadnntp.GetArticleCommonByMsgID(&sp.PSQLIB, nc, w, cs, msgid)
}

// ARTICLE/HEAD/BODY/STAT by PostNum
func (sp *PSQLIB) GetArticleFullByNum(
	w Responder, cs *ConnState, num uint64) bool {

	nc := &pireadnntp.FullNNTPCopyer{W: w}
	return pireadnntp.GetArticleCommonByNum(&sp.PSQLIB, nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleHeadByNum(
	w Responder, cs *ConnState, num uint64) bool {

	nc := &pireadnntp.HeadNNTPCopyer{W: w}
	return pireadnntp.GetArticleCommonByNum(&sp.PSQLIB, nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleBodyByNum(
	w Responder, cs *ConnState, num uint64) bool {

	nc := &pireadnntp.BodyNNTPCopyer{W: w}
	return pireadnntp.GetArticleCommonByNum(&sp.PSQLIB, nc, w, cs, num)
}
func (sp *PSQLIB) GetArticleStatByNum(
	w Responder, cs *ConnState, num uint64) bool {

	nc := &pireadnntp.StatNNTPCopyer{W: w}
	return pireadnntp.GetArticleCommonByNum(&sp.PSQLIB, nc, w, cs, num)
}

// ARTICLE/HEAD/BODY/STAT by current
func (sp *PSQLIB) GetArticleFullByCurr(w Responder, cs *ConnState) bool {
	nc := &pireadnntp.FullNNTPCopyer{W: w}
	return pireadnntp.getArticleCommonByCurr(&sp.PSQLIB, nc, w, cs)
}
func (sp *PSQLIB) GetArticleHeadByCurr(w Responder, cs *ConnState) bool {
	nc := &pireadnntp.HeadNNTPCopyer{W: w}
	return pireadnntp.getArticleCommonByCurr(&sp.PSQLIB, nc, w, cs)
}
func (sp *PSQLIB) GetArticleBodyByCurr(w Responder, cs *ConnState) bool {
	nc := &pireadnntp.BodyNNTPCopyer{W: w}
	return pireadnntp.getArticleCommonByCurr(&sp.PSQLIB, nc, w, cs)
}
func (sp *PSQLIB) GetArticleStatByCurr(w Responder, cs *ConnState) bool {
	nc := &pireadnntp.StatNNTPCopyer{W: w}
	return pireadnntp.getArticleCommonByCurr(&sp.PSQLIB, nc, w, cs)
}

// navigation
func (sp *PSQLIB) SelectGroup(w Responder, cs *ConnState, group []byte) bool {
    return pireadnntp.SelectGroup(&sp.PSQLIB, w, cs, group)
}
func (sp *PSQLIB) SelectAndListGroup(
	w Responder, cs *ConnState, group []byte, rmin, rmax int64) bool {
        
    return pireadnntp.SelectAndListGroup(&sp.PSQLIB, w, cs, group, rmin, rmax)
}
func (sp *PSQLIB) SelectNextArticle(w Responder, cs *ConnState) {
    return pireadnntp.SelectNextArticle(&sp.PSQLIB, w, cs)
}
func (sp *PSQLIB) SelectPrevArticle(w Responder, cs *ConnState) {
    return pireadnntp.SelectPrevArticle(&sp.PSQLIB, w, cs)
}

// listings
func (sp *PSQLIB) ListNewNews(
	aw AbstractResponder, wildmat []byte, qt time.Time) {
    
    return pireadnntp.ListNewNews(&sp.PSQLIB, aw, wildmat, qt)
}
func (sp *PSQLIB) ListNewGroups(aw AbstractResponder, qt time.Time) {
    return pireadnntp.ListNewGroups(&sp.PSQLIB, aw, qt)
}
func (sp *PSQLIB) ListActiveGroups(aw AbstractResponder, wildmat []byte) {
    return pireadnntp.ListActiveGroups(&sp.PSQLIB, aw, wildmat)
}
func (sp *PSQLIB) ListNewsgroups(aw AbstractResponder, wildmat []byte) {
    return pireadnntp.ListNewsgroups(&sp.PSQLIB, aw, wildmat)
}

// over stuff
func (sp *PSQLIB) GetOverByMsgID(
	w Responder, cs *ConnState, msgid TCoreMsgID) bool {
    
    return pireadnntp.GetOverByMsgID(&sp.PSQLIB, w, cs, msgid)
}
func (sp *PSQLIB) GetOverByRange(
	w Responder, cs *ConnState, rmin, rmax int64) bool {
    
    return pireadnntp.GetOverByRange(&sp.PSQLIB, w, cs, rmin, rmax)
}
func (sp *PSQLIB) GetXOverByRange(
	w Responder, cs *ConnState, rmin, rmax int64) bool {

	return pireadnntp.GetOverByRange(&sp.PSQLIB, w, cs, rmin, rmax)
}
func (sp *PSQLIB) GetOverByCurr(w Responder, cs *ConnState) bool {
    return pireadnntp.GetOverByCurr(&sp.PSQLIB, w, cs)
}

// hdr stuff
func (sp *PSQLIB) GetHdrByMsgID(
	w Responder, cs *ConnState, hdr []byte, msgid TCoreMsgID) bool {

	return pireadnntp.CommonGetHdrByMsgID(
        &sp.PSQLIB, w, cs, hdr, msgid, true)
}
func (sp *PSQLIB) GetHdrByRange(
	w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool {

	return pireadnntp.CommonGetHdrByRange(
        &sp.PSQLIB, w, cs, hdr, rmin, rmax, true)
}
func (sp *PSQLIB) GetHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool {
	return sp.CommonGetHdrByCurr(&sp.PSQLIB, w, cs, hdr, true)
}
func (sp *PSQLIB) GetXHdrByMsgID(
	w Responder, hdr []byte, msgid TCoreMsgID) bool {

	return pireadnntp.CommonGetHdrByMsgID(
        &sp.PSQLIB, w, nil, hdr, msgid, false)
}
func (sp *PSQLIB) GetXHdrByRange(
	w Responder, cs *ConnState, hdr []byte, rmin, rmax int64) bool {

	return sp.CommonGetHdrByRange(
        &sp.PSQLIB, w, cs, hdr, rmin, rmax, false)
}
func (sp *PSQLIB) GetXHdrByCurr(w Responder, cs *ConnState, hdr []byte) bool {
	return pireadnntp.CommonGetHdrByCurr(&sp.PSQLIB, w, cs, hdr, false)
}
