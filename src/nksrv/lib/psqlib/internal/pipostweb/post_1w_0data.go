package pipostweb

import (
	"database/sql"

	"github.com/lib/pq"

	"nksrv/lib/ibref_nntp"
	"nksrv/lib/mail/form"
	"nksrv/lib/mailib"
	"nksrv/lib/psqlib/internal/pibase"
	"nksrv/lib/psqlib/internal/pibaseweb"
	"nksrv/lib/psqlib/internal/pipostbase"
)

type (
	boardID = pibase.TBoardID
	postID  = pibase.TPostID
)

// info for thumbnail tmpfile location and intended final filename
type wp_thumbMove = mailib.TThumbInfo

// board/thread/reply
type wp_btr struct {
	board   string
	thread  string
	isReply bool
}

type postWebContext struct {
	pipostbase.PostCommonContext

	f form.Form

	wp_btr

	xf       webInputFields
	postOpts PostOptions

	wp_dbinfo

	pInfo     mailib.PostInfo
	isctlgrp  bool
	pubkeystr string
	srefs     []ibref_nntp.Reference
	irefs     []ibref_nntp.Index

	msgfn string // full filename of inner msg (if doing primitive signing)
}

type wp_dbinfo struct {
	bid        boardID                    // board being posted into
	tid        sql.NullInt64              // thread id if replying to thread
	ref        sql.NullString             // if replying, referenced msgid
	postLimits pibaseweb.SubmissionLimits // post limits applying for this transaction
	opdate     pq.NullTime                // date of OP for validity checking
}
