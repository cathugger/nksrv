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
type wp_thumbMove struct {
	FullTmpName string
	RelDestName string
}

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

	msgfn      string // full filename of inner msg (if doing primitive signing)
	thumbMoves []wp_thumbMove

	src_pending string // full dir name without slash of pending dir in src
	thm_pending string // full dir name without slash of pending dir in thm
}

type wp_dbinfo struct {
	bid        boardID
	tid        sql.NullInt64
	ref        sql.NullString
	postLimits submissionLimits
	opdate     pq.NullTime
}
