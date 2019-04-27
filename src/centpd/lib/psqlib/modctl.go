package psqlib

import (
	"database/sql"

	mm "centpd/lib/minimail"
)

type modCtlCmdFunc func(sp *PSQLIB, tx *sql.Tx, gpid postID, selfid, ref FullMsgIDStr, cmd string, args []string) (err error)

var modCtlCmds = map[string]modCtlCmdFunc{
	"delete": func(sp *PSQLIB, tx *sql.Tx, gpid postID, selfid, ref FullMsgIDStr, cmd string, args []string) (err error) {
		if len(args) < 1 {
			return
		}
		fmsgids := FullMsgIDStr(args[0])
		if !mm.ValidMessageIDStr(fmsgids) || fmsgids == selfid || fmsgids == ref {
			return
		}

		// TODO add ban

		err = sp.banByMsgID(tx, cutMsgID(fmsgids), gpid, "TODO")
		if err != nil {
			return
		}

		return
	},
}
