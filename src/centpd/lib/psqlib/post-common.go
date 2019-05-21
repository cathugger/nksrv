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
	ust := tx.Stmt(sp.st_prep[st_web_set_mod_priv])
	// do key update
	var modid int64
	// this probably should lock relevant row.
	// that should block reads of this row I think?
	// which would mean no further new mod posts for this key
	err = ust.QueryRow(pubkeystr, newpriv.String()).Scan(&modid)
	if err != nil {
		if err == sql.ErrNoRows {
			// we changed nothing so return now
			return nil
		}
		return sp.sqlError("st_web_set_mod_priv queryrowscan", err)
	}
	/*
	zbp.g_p_id,
	zbp.b_id,
	zbp.b_p_id,
	yb.b_name,
	yp.msgid,
	ypp.msgid,
	yf.fname
	*/
	xst := tx.Stmt(sp.st_prep[st_web_fetch_and_clear_mod_msgs])
	xst.Query(modid)
	// TODO finish

	// read msgs of mod
	// for each, clear effect of message, then parse message and apply actions

	return
}
