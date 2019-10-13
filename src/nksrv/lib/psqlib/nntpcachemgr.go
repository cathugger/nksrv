package psqlib

import (
	"database/sql"
	"io"
	"os"

	"golang.org/x/crypto/blake2s"

	ht "nksrv/lib/hashtools"
)

type nntpidinfo struct {
	bpid postID
	gpid postID
}

type nntpcachemgr struct {
	sp *PSQLIB
}

func (mgr nntpcachemgr) MakeFilename(id string) string {
	// id can contain invalid chars like /
	// we could just base32 id itself but that would allow it to grow over common file name limit of 255
	// so do blake2s
	idsum := blake2s.Sum256(unsafeStrToBytes(id))
	enc := ht.LowerBase32Enc.EncodeToString(idsum[:])
	return mgr.sp.nntpfs.Main() + enc + ".eml"
}

func (mgr nntpcachemgr) NewTempFile() (*os.File, error) {
	return mgr.sp.nntpfs.TempFile("", "")
}

func (mgr nntpcachemgr) Generate(
	w io.Writer, objid string, objinfo interface{}) error {

	x := objinfo.(nntpidinfo)
	return mgr.sp.nntpGenerate(w, CoreMsgIDStr(objid), x.gpid)
}

func (sp *PSQLIB) nntpObtainItemByMsgID(
	w nntpCopyer, cs *ConnState, msgid CoreMsgIDStr) error {

	cb_bid := currSelectedGroupID(cs)

	var p_bid boardID
	var p_bpid postID
	var p_gpid postID
	var p_isbanned bool

	err := sp.st_prep[st_nntp_article_num_by_msgid].
		QueryRow(string(msgid), cb_bid).
		Scan(&p_bid, &p_bpid, &p_gpid, &p_isbanned)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.sqlError("posts row query scan", err)
	}
	if p_isbanned {
		// we could signal this in some other way later maybe
		return errNotExist
	}

	// this kind of query should never modify current article ID

	cb_bpid := bpidIfGroupEq(cb_bid, p_bid, p_bpid)

	return sp.nntpObtainItemOrStat(w, cb_bpid, msgid, p_gpid)
}

func (sp *PSQLIB) nntpObtainItemByNum(
	w nntpCopyer, cs *ConnState, num uint64) error {

	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		return errNoBoardSelected
	}

	var p_msgid CoreMsgIDStr
	var p_gpid postID

	err := sp.st_prep[st_nntp_article_msgid_by_num].
		QueryRow(gs.bid, num).
		Scan(&p_msgid, &p_gpid)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.sqlError("posts row query scan", err)
	}

	// this kind of query modifies current article ID
	// therefore pass state to copyer so it can set it
	w.SetGroupState(gs)

	return sp.nntpObtainItemOrStat(w, num, p_msgid, p_gpid)
}

func (sp *PSQLIB) nntpObtainItemByCurr(w nntpCopyer, cs *ConnState) error {
	gs := getGroupState(cs)
	if !isGroupSelected(gs) {
		return errNoBoardSelected
	}
	if gs.bpid <= 0 {
		return errNotExist
	}

	var msgid CoreMsgIDStr
	var gpid postID

	err := sp.st_prep[st_nntp_article_msgid_by_num].
		QueryRow(gs.bid, gs.bpid).
		Scan(&msgid, &gpid)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNotExist
		}
		return sp.sqlError("posts row query scan", err)
	}

	// current article ID isn't to be modified because it'd be the same

	return sp.nntpObtainItemOrStat(w, gs.bpid, msgid, gpid)
}

func (sp *PSQLIB) nntpObtainItemOrStat(
	w nntpCopyer, bpid postID, msgid CoreMsgIDStr, gpid postID) error {

	nii := nntpidinfo{bpid: bpid, gpid: gpid}

	if _, ok := w.(*statNNTPCopyer); !ok {
		return sp.nntpce.ObtainItem(w, string(msgid), nii)
	} else {
		// interface abuse
		_, err := w.CopyFrom(nil, string(msgid), nii)
		return err
	}
}
