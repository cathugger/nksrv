package pipostnntp

import (
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
	FRef TFullMsgIDStr
}

type postNNTPContext struct {
	postCommonContext

	sp     *PSQLIB
	pi     mailib.PostInfo
	H      mail.HeaderMap
	info   nntpParsedInfo
	isSage bool

	tmpfns []string
}
