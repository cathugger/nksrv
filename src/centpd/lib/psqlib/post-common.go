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

func (sp *PSQLIB) registeredMod(tx *sql.Tx, pubkeystr string) (modid int64, priv ModPriv, err error) {
	var privstr string
	st := tx.Stmt(sp.st_prep[st_web_autoregister_mod])
	x := 0
	for {
		err = st.QueryRow(pubkeystr).Scan(&modid, &privstr)
		if err != nil {
			if err == sql.ErrNoRows && x < 100 {
				x++
				continue
			}
			err = sp.sqlError("st_web_autoregister_mod queryrowscan", err)
			return
		}
		priv = StringToModPriv(privstr)
		return
	}
}

func (sp *PSQLIB) setModPriv(tx *sql.Tx, pubkeystr string, newpriv ModPriv) (err error) {
	//ust := tx.Stmt(sp.st_prep[st_web_set_mod_priv])
	// do key update

	// if unchanged, return
	// read msgs of mod
	// for each, clear effect of message, then parse message and apply actions

	return
}
