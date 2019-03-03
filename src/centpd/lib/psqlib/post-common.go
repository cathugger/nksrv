package psqlib

import (
	"database/sql"

	"centpd/lib/thumbnailer"
)

func (sp *PSQLIB) pickThumbPlan(isReply, isSage bool) thumbnailer.ThumbPlan {
	if !isReply {
		return sp.tplan_thread
	} else if !isSage {
		return sp.tplan_reply
	} else {
		return sp.tplan_sage
	}
}

func (sp *PSQLIB) registeredMod(pubkeystr string) (modid int64, priv ModPriv, err error) {
	var privstr string
	x := 0
	for {
		err = sp.st_prep[st_Web_autoregister_mod].QueryRow(pubkeystr).Scan(&modid, &privstr)
		if err != nil {
			if err == sql.ErrNoRows && x < 100 {
				x++
				continue
			}
			err = sp.sqlError("st_Web_autoregister_mod queryrowscan", err)
			return
		}
		priv = StringToModPriv(privstr)
		return
	}
}
