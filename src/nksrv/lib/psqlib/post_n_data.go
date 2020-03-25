package psqlib

import (
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mailib"
)

type insertSqlInfo struct {
	postLimits submissionLimits
	threadOpts threadOptions
	tid        postID
	bid        boardID
	isReply    bool
	refSubject string
}

type nntpParsedInfo struct {
	insertSqlInfo
	mailib.ParsedMessageInfo
	FRef FullMsgIDStr
}

type nntpPostCtx struct {
	sp     *PSQLIB
	H      mail.Headers
	info   nntpParsedInfo
	isSage bool
}
