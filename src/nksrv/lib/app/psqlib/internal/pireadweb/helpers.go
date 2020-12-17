package pireadweb

import (
	"nksrv/lib/app/psqlib/internal/pibase"
	ib0 "nksrv/lib/app/webib0"
	"nksrv/lib/mail"
)

type (
	boardID = pibase.TBoardID
	postID  = pibase.TPostID
)

func ensureThumb(
	sp *pibase.PSQLIB,
	t ib0.IBThumbInfo, fname, ftype string) ib0.IBThumbInfo {

	if t.ID == "" {
		t.Alt, t.Width, t.Height = sp.AltThumber.GetAltThumb(fname, ftype)
	}
	return t
}

func webCleanHeaders(h mail.HeaderMap) {
	delete(h, "Message-ID")
	delete(h, "MIME-Version")
	delete(h, "Content-Type")
}
