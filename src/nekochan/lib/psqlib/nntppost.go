package psqlib

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path"

	"nekochan/lib/mail"
	"nekochan/lib/nntp"
)

// mandatory headers for transmission. POST uses separate system
var hdrNNTPMandatory = [...]struct {
	h string // header
	o bool   // optional (allow absence?)
}{
	// NetNews stuff specified in {RFC 5536}
	{"Message-ID", true}, // special handling
	{"From", false},
	{"Date", false},
	{"Newsgroups", false},
	{"Path", true},    // more lax than {RFC 5536}
	{"Subject", true}, // more lax than {RFC 5536} (no subject is much better than "none")

	// {RFC 5322}
	{"Sender", true},
	{"Reply-To", true},
	{"To", true},
	{"Cc", true},
	{"Bcc", true},
	{"In-Reply-To", true},
	{"References", true},

	// some extras we process
	{"Injection-Date", true},
	{"NNTP-Posting-Date", true},
}

func validMsgID(s FullMsgIDStr) bool {
	return nntp.ValidMessageID(unsafeStrToBytes(string(s)))
}

func reservedMsgID(s FullMsgIDStr) bool {
	return nntp.ReservedMessageID(unsafeStrToBytes(string(s)))
}

func cutMsgID(s FullMsgIDStr) CoreMsgIDStr {
	return CoreMsgIDStr(unsafeBytesToStr(
		nntp.CutMessageID(unsafeStrToBytes(string(s)))))
}

func (sp *PSQLIB) acceptNewsgroupArticle(group string) error {
	// TODO ability to autoadd group?
	var dummy int
	q := `SELECT 1 FROM ib0.boards WHERE bname = $1 LIMIT 1`
	err := sp.db.DB.QueryRow(q, group).Scan(&dummy)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoSuchBoard
		} else {
			return sp.sqlError("board existence query scan", err)
		}
	} else {
		// board exists so accept it
		return nil
	}
}

func (sp *PSQLIB) nntpCheckArticleExists(
	w Responder, unsafe_sid CoreMsgIDStr) (exists bool, err error) {

	var dummy int
	q := "SELECT 1 FROM ib0.posts WHERE msgid = $1 LIMIT 1"
	err = sp.db.DB.QueryRow(q, string(unsafe_sid)).Scan(&dummy)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, sp.sqlError("article existence query scan", err)
		}
		return false, nil
	} else {
		return true, nil
	}
}

type nntpParsedInfo struct {
	MessageID  CoreMsgIDStr
	PostedDate int64
	Newsgroup  string
}

var (
	nntpIncomingTempDir = "_tin"
	nntpIncomingDir     = "_in"
)

func (sp *PSQLIB) nntpSendIncomingArticle(
	name string, H mail.Headers, info nntpParsedInfo) {
	// TODO
}

// + iok: 335{ResSendArticleToBeTransferred} ifail: 435{ResTransferNotWanted[false]} 436{ResTransferFailed}
// cok: 235{ResTransferSuccess} cfail: 436{ResTransferFailed} 437{ResTransferRejected}
func (sp *PSQLIB) HandleIHave(
	w Responder, cs *ConnState, ro nntp.ReaderOpener, msgid CoreMsgID) bool {

	var err error

	unsafe_sid := unsafeCoreMsgIDToStr(msgid)

	// check if we already have it
	exists, err := sp.nntpCheckArticleExists(w, unsafe_sid)
	if err != nil {
		w.ResInternalError(err)
		return true
	}
	if exists {
		// article exists
		return false
	}

	w.ResSendArticleToBeTransferred()
	r := ro.OpenReader()

	var mh mail.MessageHead
	defer func() {
		mh.Close()
		// CAUTION err MUST be set for this to work
		if err != nil {
			r.Discard(-1)
		}
	}()
	mh, err = mail.ReadHeaders(r, 2<<20)
	if err != nil {
		err = fmt.Errorf("failed reading headers: %v", err)
		w.ResTransferRejected(err)
		return true
	}

	info, ok := sp.nntpDigestTransferHead(w, mh.H, unsafe_sid)
	if !ok {
		return true
	}

	// TODO file should start with current timestamp/increasing counter
	f, err := sp.nntpfs.NewFile(nntpIncomingTempDir, "", ".eml")
	if err != nil {
		err = fmt.Errorf("error on making temporary file: %v", err)
		w.ResInternalError(err)
		return true
	}
	defer func() {
		if err != nil {
			n := f.Name()
			f.Close()
			os.Remove(n)
		}
	}()

	err = mail.WriteHeaders(f, mh.H, false)
	if err != nil {
		if err == mail.ErrHeaderLineTooLong {
			w.ResTransferRejected(err)
		} else {
			err = fmt.Errorf("error writing headers: %v", err)
			w.ResInternalError(err)
		}
		return true
	}

	_, err = fmt.Fprintf(f, "\n")
	if err != nil {
		err = fmt.Errorf("error writing body: %v", err)
		w.ResInternalError(err)
		return true
	}

	// TODO make limit configurable
	const limit = 256 << 20
	n, err := io.CopyN(f, mh.B, limit+1)
	if err != nil {
		err = fmt.Errorf("error writing body: %v", err)
		w.ResInternalError(err)
		return true
	}
	if n > limit {
		err = fmt.Errorf("message body too large, up to %d allowed", limit)
		w.ResTransferRejected(err)
	}
	err = f.Close()
	if err != nil {
		err = fmt.Errorf("error writing body: %v", err)
		w.ResInternalError(err)
		return true
	}

	newname := path.Join(sp.nntpfs.Main()+nntpIncomingDir, path.Base(f.Name()))
	err = os.Rename(f.Name(), newname)
	if err != nil {
		err = sp.sqlError("incoming file move", err)
		w.ResInternalError(err)
		return true
	}

	sp.nntpSendIncomingArticle(newname, mh.H, info)

	return true
}

// + ok: 238{ResArticleWanted} fail: 431{ResArticleWantLater} 438{ResArticleNotWanted[false]}
func (sp *PSQLIB) HandleCheck(w Responder, cs *ConnState, msgid CoreMsgID) bool {
	// TODO
	w.ResInternalError(fmt.Errorf("unimplemented"))
	return true
}

// + ok: 239{ResArticleTransferedOK} 439{ResArticleRejected[false]}
func (sp *PSQLIB) HandleTakeThis(w Responder, cs *ConnState, r nntp.ArticleReader, msgid CoreMsgID) bool {
	w.ResInternalError(fmt.Errorf("unimplemented"))
	r.Discard(-1)
	return true
}

/*
	h, e := mail.ReadHeaders(r, 2<<20)
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
*/
