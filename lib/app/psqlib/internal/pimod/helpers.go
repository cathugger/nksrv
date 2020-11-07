package pimod

import (
	. "nksrv/lib/app/psqlib/internal/pibasenntp"
	"nksrv/lib/nntp"
)

func validMsgID(s TFullMsgIDStr) bool {
	return nntp.ValidMessageID(unsafeStrToBytes(string(s)))
}

func cutMsgID(s TFullMsgIDStr) TCoreMsgIDStr {
	return TCoreMsgIDStr(unsafeBytesToStr(
		nntp.CutMessageID(unsafeStrToBytes(string(s)))))
}
