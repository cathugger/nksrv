package psqlib

import (
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	. "nksrv/lib/logx"
	"nksrv/lib/mailib"
	"nksrv/lib/thumbnailer"
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

func (sp *PSQLIB) registeredMod(
	tx *sql.Tx, pubkeystr string) (modid int64, priv ModPriv, err error) {

	// mod posts MAY later come back and want more of things in this table (if they eval/GC modposts)
	// at which point we're fucked because moddel posts also will exclusively block files table
	// and then we won't be able to insert into it..
	_, err = tx.Exec("LOCK ib0.modlist IN EXCLUSIVE MODE")
	if err != nil {
		err = sp.sqlError("lock ib0.modlist query", err)
	}

	sp.log.LogPrintf(DEBUG, "REGMOD %s done locking ib0.modlist", pubkeystr)

	var privstr string
	st := tx.Stmt(sp.st_prep[st_web_autoregister_mod])
	x := 0
	for {
		err = st.QueryRow(pubkeystr).Scan(&modid, &privstr)
		if err != nil {

			if err == sql.ErrNoRows && x < 100 {

				x++

				sp.log.LogPrintf(DEBUG, "REGMOD %s retry", pubkeystr)

				continue
			}
			err = sp.sqlError("st_web_autoregister_mod queryrowscan", err)
			return
		}
		priv, _ = StringToModPriv(privstr)
		return
	}
}

func (sp *PSQLIB) setModPriv(
	tx *sql.Tx, pubkeystr string, newpriv ModPriv, indelmsgids delMsgIDState) (
	outdelmsgids delMsgIDState, err error) {

	outdelmsgids = indelmsgids

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
			sp.log.LogPrintf(DEBUG, "setmodpriv: %s priv unchanged", pubkeystr)
			err = nil
			return
		}
		err = sp.sqlError("st_web_set_mod_priv queryrowscan", err)
		return
	}

	sp.log.LogPrintf(DEBUG,
		"setmodpriv: %s priv changed, modid %d", pubkeystr, modid)

	srcdir := sp.src.Main()
	xst := tx.Stmt(sp.st_prep[st_web_fetch_and_clear_mod_msgs])

	offset := uint64(0)

	type idt struct {
		bid  boardID
		bpid postID
	}
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
	lastx := idt{0, 0}

	for {
		var rows *sql.Rows
		rows, err = xst.Query(modid, offset)
		if err != nil {
			err = sp.sqlError("st_web_fetch_and_clear_mod_msgs query", err)
			return
		}

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
				err = sp.sqlError("st_web_fetch_and_clear_mod_msgs rows scan", err)
				return
			}

			if lastx != p.xid {
				lastx = p.xid
				p.ref = ref.String
				p.txtidx = uint32(txtidx.Int64)
				posts = append(posts, p)
			}
			pp := &posts[len(posts)-1]
			if fname.String != "" {
				pp.files = append(pp.files, srcdir+fname.String)
			}
		}
		if err = rows.Err(); err != nil {
			err = sp.sqlError("st_web_fetch_and_clear_mod_msgs rows it", err)
			return
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

			sp.log.LogPrintf(DEBUG,
				"setmodpriv: executing <%s> from board[%s]",
				posts[i].msgid, posts[i].bname)

			outdelmsgids, err = sp.execModCmd(
				tx, posts[i].gpid, posts[i].xid.bid, posts[i].xid.bpid, modid,
				newpriv, pi, posts[i].files, pi.MessageID,
				CoreMsgIDStr(posts[i].ref), outdelmsgids)
			if err != nil {
				return
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
			_ = tx.Rollback()
		}
	}()

	var delmsgids delMsgIDState
	defer sp.cleanDeletedMsgIDs(delmsgids)

	for _, s := range mods {
		sp.log.LogPrintf(INFO, "setmodpriv %s %s", s, newpriv.String())

		delmsgids, err = sp.setModPriv(tx, s, newpriv, delmsgids)
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
