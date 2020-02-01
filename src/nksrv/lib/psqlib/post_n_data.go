package psqlib

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	au "nksrv/lib/asciiutils"
	"nksrv/lib/date"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/ibref_nntp"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mailib"
	"nksrv/lib/mailibsign"
	"nksrv/lib/nntp"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
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
	sp *PSQLIB
	H mail.Headers
	info nntpParsedInfo
	isSage bool
}
