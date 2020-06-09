package pireadweb

// implements web imageboard interface v0

import (
	"nksrv/lib/mail"
	ib0 "nksrv/lib/webib0"
)

// functionality

func (sp *PSQLIB) ensureThumb(
	t ib0.IBThumbInfo, fname, ftype string) ib0.IBThumbInfo {

	if t.ID == "" {
		t.Alt, t.Width, t.Height = sp.altthumb.GetAltThumb(fname, ftype)
	}
	return t
}

func webCleanHeaders(h mail.HeaderMap) {
	delete(h, "Message-ID")
	delete(h, "MIME-Version")
	delete(h, "Content-Type")
}
