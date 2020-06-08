package mailib

import "nksrv/lib/minimail"

type TFullMsgID = minimail.TFullMsgID // msgid with < and >
type TCoreMsgID = minimail.TCoreMsgID // msgid without < and >
type TFullMsgIDStr = minimail.TFullMsgIDStr
type TCoreMsgIDStr = minimail.TCoreMsgIDStr

type ParsedMessageInfo struct {
	FullMsgIDStr TFullMsgIDStr
	PostedDate   int64
	Newsgroup    string
}
