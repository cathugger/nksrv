package mailib

import "nksrv/lib/minimail"

type FullMsgID = minimail.FullMsgID // msgid with < and >
type CoreMsgID = minimail.CoreMsgID // msgid without < and >
type FullMsgIDStr = minimail.FullMsgIDStr
type CoreMsgIDStr = minimail.CoreMsgIDStr

type ParsedMessageInfo struct {
	FullMsgIDStr FullMsgIDStr
	PostedDate   int64
	Newsgroup    string
}
