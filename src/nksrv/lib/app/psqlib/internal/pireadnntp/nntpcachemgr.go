package pireadnntp

import (
	"database/sql"
	"io"
	"os"

	"golang.org/x/crypto/blake2s"

	"nksrv/lib/app/psqlib/internal/pibase"
	ht "nksrv/lib/utils/hashtools"
)

type nntpidinfo struct {
	bpid postID
	gpid postID
}

type NNTPCacheMgr struct {
	*pibase.PSQLIB
}

func (mgr NNTPCacheMgr) MakeFilename(id string) string {
	// id can contain invalid chars like /
	// we could just base32 id itself but that would allow it to grow over common file name limit of 255
	// so do blake2s
	// TODO we should just store cache key in DB itself
	idsum := blake2s.Sum256(unsafeStrToBytes(id))
	enc := ht.LowerBase32Enc.EncodeToString(idsum[:])
	return mgr.NNTPFS.Main() + enc + ".eml"
}

func (mgr NNTPCacheMgr) NewTempFile() (*os.File, error) {
	return mgr.NNTPFS.NewFile("tmp", "", "")
}

func (mgr NNTPCacheMgr) Generate(
	w io.Writer, objid string, objinfo interface{}) error {

	x := objinfo.(nntpidinfo)
	return nntpGenerate(mgr.PSQLIB, w, TCoreMsgIDStr(objid), x.gpid)
}

func nntpObtainItemByMsgID(
	sp *pibase.PSQLIB, w nntpCopyer, cs *ConnState, msgid TCoreMsgIDStr) error {

	cb_bid := currSelectedGroupID(cs)

	var p_bid boardID
	var p_bpid postID
	var p_gpid postID
	var p_isbanned bool

	err := sp.StPrep[pibase.St_nntp_article_num_by_msgid].
		QueryRow(string(msgid), cb_bid).
		Scan(&p_bid, &p_bpid, &p_gpid, &p_isbanned)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.SQLError("posts row query scan", err)
	}
	if p_isbanned {
		// we could signal this in some other way later maybe
		return errNotExist
	}

	// this kind of query should never modify current article ID

	cb_bpid := bpidIfGroupEq(cb_bid, p_bid, p_bpid)

	return nntpObtainItemOrStat(sp, w, cb_bpid, msgid, p_gpid)
}

func nntpObtainItemByNum(
	sp *pibase.PSQLIB, w nntpCopyer, cs *ConnState, num uint64) error {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		return errNoBoardSelected
	}

	var p_msgid TCoreMsgIDStr
	var p_gpid postID

	err := sp.StPrep[pibase.St_nntp_article_msgid_by_num].
		QueryRow(gs.bid, num).
		Scan(&p_msgid, &p_gpid)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.SQLError("posts row query scan", err)
	}

	// this kind of query modifies current article ID
	// therefore pass state to copyer so it can set it
	w.SetGroupState(gs)

	return nntpObtainItemOrStat(sp, w, num, p_msgid, p_gpid)
}

func nntpObtainItemByCurr(
	sp *pibase.PSQLIB, w nntpCopyer, cs *ConnState) error {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		return errNoBoardSelected
	}
	if gs.bpid <= 0 {
		return errNotExist
	}

	var msgid TCoreMsgIDStr
	var gpid postID

	err := sp.StPrep[pibase.St_nntp_article_msgid_by_num].
		QueryRow(gs.bid, gs.bpid).
		Scan(&msgid, &gpid)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.SQLError("posts row query scan", err)
	}

	// current article ID isn't to be modified because it'd be the same

	return nntpObtainItemOrStat(sp, w, gs.bpid, msgid, gpid)
}

func nntpObtainItemOrStat(
	sp *pibase.PSQLIB, w nntpCopyer,
	bpid postID, msgid TCoreMsgIDStr, gpid postID) error {

	nii := nntpidinfo{bpid: bpid, gpid: gpid}

	if _, ok := w.(*StatNNTPCopyer); !ok {
		return sp.NNTPCE.ObtainItem(w, string(msgid), nii)
	} else {
		// interface abuse
		_, err := w.CopyFrom(nil, string(msgid), nii)
		return err
	}
}
