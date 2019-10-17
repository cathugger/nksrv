package psqlib

import (
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	xtypes "github.com/jmoiron/sqlx/types"

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
	tx *sql.Tx, pubkeystr string) (
	modid uint64, hascap bool, modCap ModCap, modBoardCap ModBoardCap,
	err error) {

	// mod posts MAY later come back and want more of things in this table (if they eval/GC modposts)
	// at which point we're fucked because moddel posts also will exclusively block files table
	// and then we won't be able to insert into it..
	_, err = tx.Exec("LOCK ib0.modlist IN EXCLUSIVE MODE")
	if err != nil {
		err = sp.sqlError("lock ib0.modlist query", err)
		return
	}

	sp.log.LogPrintf(DEBUG, "REGMOD %s done locking ib0.modlist", pubkeystr)

	st := tx.Stmt(sp.st_prep[st_mod_autoregister_mod])
	x := 0
	for {

		var mcap sql.NullString
		var mbcap map[string]string, mbcapj xtypes.JSONText
		var mdpriv sql.NullInt32
		var mbdpriv map[string]string, mbdprivj xtypes.JSONText

		err = st.QueryRow(pubkeystr).Scan(
			&modid, &mcap, &mbcapj, &mdpriv, &mbdprivj)

		if err != nil {

			if err == sql.ErrNoRows && x < 100 {

				x++

				sp.log.LogPrintf(DEBUG, "REGMOD %s retry", pubkeystr)

				continue
			}

			err = sp.sqlError("st_web_autoregister_mod queryrowscan", err)
			return
		}

		err = mbcapj.Unmarshal(&mbcap)
		if err != nil { panic("mbcap.Unmarshal") }

		err = mbdprivj.Unmarshal(&mbdpriv)
		if err != nil { panic("mbdpriv.Unmarshal") }

		hascap = mcap.Valid || len(mbcap) != 0 ||
			mdpriv.Valid || len(mbdpriv) != 0

		if mcap.Valid {
			modCap.Cap = StrToCap(mcap.String)
		}
		if mdpriv.Valid {
			modCap.DPriv = int16(mdpriv.Int32 & 0x7Fff)
		}

		modBoardCap = make(ModBoardCap)
		modBoardCap.takeIn(mbcap, mbdpriv)

		return
	}
}

func (sp *PSQLIB) setModCap(
	tx *sql.Tx, pubkeystr, group string, newcap ModCap) (err error) {

	ust := tx.Stmt(sp.st_prep[st_mod_set_mod_priv])
	// do key update
	var modid uint64
	// this probably should lock relevant row.
	// that should block reads of this row I think?
	// which would mean no further new mod posts for this key
	err = ust.QueryRow(
		pubkeystr,
		sql.NullString{
			String: group,
			Valid: group != "",
		},
		newcap.String(),
		sql.NullInt32{
			Int32: int32(newcap.DPriv),
			Valid: newcap.DPriv >= 0,
		}).
		Scan(&modid)

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

	return
}

func (sp *PSQLIB) xxxx(
	tx *sql.Tx, _in_delmsgids delMsgIDState) (
	out_delmsgids delMsgIDState, err error) {

	out_delmsgids = _in_delmsgids

	srcdir := sp.src.Main()
	xst := tx.Stmt(sp.st_prep[st_mod_fetch_and_clear_mod_msgs])

	// 666 days in the future
	off_pdate := time.Now().Add(time.Hour * 24 * 666).UTC()
	off_g_p_id := uint64(0)
	off_b_id := uint32(0)

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
		pdate   time.Time
		message string
		txtidx  uint32
		files   []string
	}
	var posts []postinfo
	lastx := idt{0, 0}

requery:
	for {
		var rows *sql.Rows
		rows, err = xst.Query(modid, off_pdate, off_g_p_id, off_b_id)
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
				&p.title, &p.pdate, &p.message, &txtidx, &fname)
			if err != nil {
				rows.Close()
				err = sp.sqlError("st_web_fetch_and_clear_mod_msgs rows scan", err)
				return
			}

			if lastx != p.xid {
				lastx = p.xid
				p.ref = ref.String
				p.txtidx = uint32(txtidx.Int64)
				p.pdate = p.pdate.UTC()
				posts = append(posts, p)
			}
			if fname.String != "" {
				pp := &posts[len(posts)-1]
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
				Date:      posts[i].pdate,
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

			var delmodids delModIDState
			var inputerr bool

			out_delmsgids, delmodids, err, inputerr = sp.execModCmd(
				tx, posts[i].gpid, posts[i].xid.bid, posts[i].xid.bpid, modid,
				newcap, pi, posts[i].files, pi.MessageID,
				CoreMsgIDStr(posts[i].ref), out_delmsgids, delmodids)

			if err != nil {

				if inputerr {
					// mod msg is just fucked at this point
					// this shouldn't happen
					// XXX should we delete this bad msg??
					sp.log.LogPrintf(ERROR,
						"setmodpriv: [proceeding anyway] inputerr while execing <%s>: %v",
						posts[i].msgid, err)
					err = nil
					continue
				}

				// return err directly
				return
			}

			if delmodids.contain(modid) {
				sp.log.LogPrintf(
					DEBUG, "setmodpriv: delmodid %d is ours, requerying", modid)
				// msg we just deleted was made by mod we just upp'd
				// that means that it may be msg in query we just made
				// it's unsafe to proceed with current cached query
				off_pdate = posts[i].pdate
				off_g_p_id = posts[i].gpid
				off_b_id = posts[i].xid.bid
				posts = posts[:0]
				continue requery
			}
		}

		if len(posts) < 4096 {
			// if less than limit that means we dont need another query
			break
		} else {
			// issue another query, there may be more data
			i := len(posts) - 1
			off_pdate = posts[i].pdate
			off_g_p_id = posts[i].gpid
			off_b_id = posts[i].xid.bid
			posts = posts[:0]
			continue requery
		}
	}

	return
}

func (sp *PSQLIB) DemoSetModCap(mods []string, newcap ModCap) {
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
	defer func() { sp.cleanDeletedMsgIDs(delmsgids) }()

	for _, s := range mods {
		sp.log.LogPrintf(INFO, "setmodpriv %s %s", s, modcap.String())

		delmsgids, err = sp.setModPriv(tx, s, modcap, delmsgids)
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

func checkFiles() {
	//
	//sp.st_prep[st_mod_load_files].
}
