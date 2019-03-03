package psqlib

import (
	"database/sql"

	mm "centpd/lib/minimail"
)

type modCtlCmdFunc func(tx *sql.Tx, gpid postID, selfid, ref FullMsgIDStr, cmd string, args []string) (err error)

var modCtlCmds = map[string]modCtlCmdFunc{
	"delete": func(tx *sql.Tx, gpid postID, selfid, ref FullMsgIDStr, cmd string, args []string) (err error) {
		if len(args) < 1 {
			return
		}
		fmsgids := FullMsgIDStr(args[0])
		if !mm.ValidMessageIDStr(fmsgids) || fmsgids == selfid || fmsgids == ref {
			return
		}

		// TODO add ban

		err = deleteByMsgID(tx, cutMsgID(fmsgids))
		if err != nil {
			return
		}

		return
	},
}
