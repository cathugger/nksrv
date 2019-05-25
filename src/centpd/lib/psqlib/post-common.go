package psqlib

import (
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	. "centpd/lib/logx"
	"centpd/lib/mailib"
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
		priv, _ = StringToModPriv(privstr)
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
	xst := tx.Stmt(sp.st_prep[st_web_fetch_and_clear_mod_msgs])
	offset := uint64(0)
	for {
		rows, err := xst.Query(modid, offset)
		if err != nil {
			return sp.sqlError("st_web_fetch_and_clear_mod_msgs query", err)
		}
		type idt struct {
			bid  boardID
			bpid postID
		}
		lastx := idt{0, 0}
		type postinfo struct {
			gpid    postID
			xid     idt
			bname   string
			msgid   string
			ref     string
			title   string
			date    time.Time
			message string
			txtidx  uint32
			files   []string
		}
		var posts []postinfo
		for rows.Next() {
			/*
				zbp.g_p_id,
				zbp.b_id,
				zbp.b_p_id,
				yb.b_name,
				yp.msgid,
				ypp.msgid,
				yp.title,
				yp.pdate,
				yp.message,
				yp.extras -> 'text_attach',
				yf.fname
			*/
			var p postinfo
			var ref, fname sql.NullString
			var txtidx sql.NullInt64

			err = rows.Scan(
				&p.gpid, &p.xid.bid, &p.xid.bpid, &p.bname, &p.msgid, &ref,
				&p.title, &p.date, &p.message, &txtidx, &fname)
			if err != nil {
				rows.Close()
				return sp.sqlError("st_web_fetch_and_clear_mod_msgs rows scan", err)
			}

			if lastx != p.xid {
				lastx = p.xid
				p.ref = ref.String
				p.txtidx = uint32(txtidx.Int64)
				posts = append(posts, p)
			}
			pp := &posts[len(posts)-1]
			if fname.String != "" {
				pp.files = append(pp.files, fname.String)
			}
		}
		if err = rows.Err(); err != nil {
			return sp.sqlError("st_web_fetch_and_clear_mod_msgs rows it", err)
		}

		for i := range posts {
			// prepare postinfo good enough for execModCmd
			pi := mailib.PostInfo{
				MessageID: CoreMsgIDStr(posts[i].msgid),
				Date:      posts[i].date.UTC(),
				MI: mailib.MessageInfo{
					Title:   posts[i].title,
					Message: posts[i].message,
				},
				E: mailib.PostExtraAttribs{
					TextAttachment: posts[i].txtidx,
				},
			}
			err = sp.execModCmd(
				tx, posts[i].gpid, posts[i].xid.bid, posts[i].xid.bpid, modid,
				newpriv, pi, posts[i].files, pi.MessageID, CoreMsgIDStr(posts[i].ref))
			if err != nil {
				return err
			}
		}

		if len(posts) < 4096 {
			// if less than limit that means we dont need another query
			break
		} else {
			// issue another query, there may be more data
			offset += uint64(len(posts))
			posts = posts[:0]
			continue
		}
	}

	return
}

func (sp *PSQLIB) DemoSetModPriv(mods []string, newpriv ModPriv) {
	var err error

	for i, s := range mods {
		if _, err = hex.DecodeString(s); err != nil {
			sp.log.LogPrintf(ERROR, "invalid modid %q", s)
			return
		}
		// we use uppercase (I forgot why)
		mods[i] = strings.ToUpper(s)
	}

	tx, err := sp.db.DB.Begin()
	if err != nil {
		err = sp.sqlError("tx begin", err)
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, s := range mods {
		sp.log.LogPrintf(INFO, "setmodpriv %s %s", s, newpriv.String())

		err = sp.setModPriv(tx, s, newpriv)
		if err != nil {
			sp.log.LogPrintf(ERROR, "%v", err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = sp.sqlError("tx commit", err)
		sp.log.LogPrintf(ERROR, "%v", err)
		return
	}
}
