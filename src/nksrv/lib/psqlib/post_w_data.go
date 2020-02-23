package psqlib

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/ed25519"

	"nksrv/lib/date"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/ibref_nntp"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mail/form"
	"nksrv/lib/mailib"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/webcaptcha"
	ib0 "nksrv/lib/webib0"
)


// info for thumbnail tmpfile location and intended final filename
type wp_thumbMove = mailib.ThumbInfo

// board/thread/reply
type wp_btr struct {
	board      string
	thread     string
	isReply    bool
}

type wp_context struct {
	sp         *PSQLIB

	f          form.Form

	wp_btr

	xf         webInputFields
	postOpts   PostOptions

	wp_dbinfo

	pInfo      mailib.PostInfo
	isctlgrp   bool
	srefs      []ibref_nntp.Reference
	irefs      []ibref_nntp.Index

	wg_TP sync.WaitGroup // for tmp->pending
	wg_PA sync.WaitGroup // for pending->active storage

	werr_mu sync.Mutex
	werr    error

	msgfn      string // full filename of inner msg (if doing primitive signing)
	thumbMoves []wp_thumbMove

	src_pending string // full dir name without slash of pending dir in src
	thm_pending string // full dir name without slash of pending dir in thm
}

func (c *wp_context) set_werr(e error) {
	c.werr_mu.Lock()
	if (c.werr == nil) != (e == nil) {
		c.werr = e
	}
	c.werr_mu.Unlock()
}

func (c *wp_context) get_werr() (e error) {
	c.werr_mu.Lock()
	e = c.werr
	c.werr_mu.Unlock()
	return
}

type wp_dbinfo struct {
	bid        boardID
	tid        sql.NullInt64
	ref        sql.NullString
	postLimits submissionLimits
	opdate     pq.NullTime
}
