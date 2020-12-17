package psqlib

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"

	"nksrv/lib/app/mailib"
	"nksrv/lib/app/psqlib/internal/pibase"
	. "nksrv/lib/utils/logx"
)

func (sp *PSQLIB) modset_processJobOnce(
	fetchMsgsOnce, fetchMsgsTotal int) (hadwork bool, err error) {

	for {
		var tx *sql.Tx
		tx, err = sp.db.DB.Begin()
		if err != nil {
			err = sp.SQLError("begin tx", err)
			return
		}

		err = sp.makeDelTables(tx)
		if err != nil {
			return
		}

		var delmsgids delMsgIDState
		hadwork, delmsgids, err =
			sp.modset_processJobOnce_tx(tx, fetchMsgsOnce, fetchMsgsTotal)

		if err == nil {
			err = tx.Commit()
			if err != nil {
				err = sp.SQLError("tx commit", err)
			}
		}
		if err != nil {
			_ = tx.Rollback()
		}

		sp.cleanDeletedMsgIDs(delmsgids)

		var dlerr psqlDeadlockError
		if errors.As(err, &dlerr) {
			// if deadlock, try again
			continue
		}

		return
	}
}

func (sp *PSQLIB) modset_processJobOnce_tx(
	tx *sql.Tx, fetchMsgsOnce, fetchMsgsTotal int) (
	hadwork bool, out_delmsgids delMsgIDState, err error) {

	srcdir := sp.src.Main()

	mcg := tx.Stmt(sp.StPrep[pibase.St_mod_joblist_modlist_changes_get])
	mcs := tx.Stmt(sp.StPrep[pibase.St_mod_joblist_modlist_changes_set])
	mcd := tx.Stmt(sp.StPrep[pibase.St_mod_joblist_modlist_changes_del])
	xfcs := tx.Stmt(sp.StPrep[pibase.St_mod_fetch_and_clear_mod_msgs_start])
	xfcc := tx.Stmt(sp.StPrep[pibase.St_mod_fetch_and_clear_mod_msgs_continue])

	var (
		j_id   uint64
		mod_id uint64

		t_date_sent sql.NullTime
		t_g_p_id    sql.NullInt64
		t_b_id      sql.NullInt32

		f modPrivFetch
	)

	err = mcg.QueryRow().Scan(
		&j_id,
		&mod_id,

		&t_date_sent,
		&t_g_p_id,
		&t_b_id,

		&f.m_g_cap,
		&f.m_b_cap_j,
		pq.Array(&f.m_g_caplvl),
		&f.m_b_caplvl_j,

		&f.mi_g_cap,
		&f.mi_b_cap_j,
		pq.Array(&f.mi_g_caplvl),
		&f.mi_b_caplvl_j)

	if err != nil {

		if err == sql.ErrNoRows {
			// overwrite err
			err = nil
			return
		}

		err = sp.SQLError("queryrowscan", err)
		return
	}

	// we got some work
	hadwork = true

	// eat caps
	f.unmarshalJSON()
	mcc := f.parse()

	numProcessed := 0
	numRequest := fetchMsgsOnce // number of rows to request

requery:
	for {
		// fetch em
		type idt struct {
			bid  boardID
			bpid postID
		}
		type postinfo struct {
			gpid      postID
			xid       idt
			bname     string
			msgid     string
			ref       string
			title     string
			date_sent time.Time
			message   string
			txtidx    uint32
			files     []string
		}
		var posts []postinfo
		lastx := idt{0, 0}

		// process rows
		var rows *sql.Rows

		if !t_g_p_id.Valid {

			sp.log.LogPrintf(
				DEBUG,
				"setmodpriv: requesting modid(%d) num(%d) start",
				mod_id, numRequest)

			rows, err = xfcs.Query(mod_id, numRequest)

		} else {

			sp.log.LogPrintf(
				DEBUG,
				"setmodpriv: requesting modid(%d) num(%d) continue (%v,%v,%v)",
				mod_id, numRequest,
				t_date_sent.Time, t_g_p_id.Int64, t_b_id.Int32)

			rows, err = xfcc.Query(mod_id, numRequest, t_date_sent, t_g_p_id, t_b_id)

		}

		if err != nil {
			err = sp.SQLError("query", err)
			return
		}

		for rows.Next() {
			var p postinfo
			var ref, fname sql.NullString
			var txtidx sql.NullInt32

			err = rows.Scan(
				&p.date_sent, &p.gpid, &p.xid.bid, &p.xid.bpid,
				&p.bname, &p.msgid, &ref,
				&p.title, &p.message, &txtidx, &fname)
			if err != nil {
				rows.Close()
				err = sp.SQLError("rows scan", err)
				return
			}

			if lastx != p.xid {
				lastx = p.xid

				p.ref = ref.String
				p.txtidx = uint32(txtidx.Int32) // NULL is same as 0
				p.date_sent = p.date_sent.UTC()

				posts = append(posts, p)
			}

			if fname.String != "" {
				pp := &posts[len(posts)-1]
				pp.files = append(pp.files, srcdir+fname.String)
			}
		}
		if err = rows.Err(); err != nil {
			err = sp.SQLError("rows it", err)
			return
		}

		// process messages
		for i := range posts {
			// prepare postinfo good enough for execModCmd
			pi := mailib.PostInfo{
				MessageID: TCoreMsgIDStr(posts[i].msgid),
				Date:      posts[i].date_sent,
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
				tx, posts[i].gpid, posts[i].xid.bid, posts[i].xid.bpid,
				mod_id, mcc,
				pi, posts[i].files, pi.MessageID,
				TCoreMsgIDStr(posts[i].ref), out_delmsgids, delmodids)

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

			t_date_sent = sql.NullTime{
				Time:  posts[i].date_sent,
				Valid: true,
			}
			t_g_p_id = sql.NullInt64{
				Int64: int64(posts[i].gpid),
				Valid: true,
			}
			t_b_id = sql.NullInt32{
				Int32: int32(posts[i].xid.bid),
				Valid: true,
			}

			// exceptional case - we axed msg of our mod_id
			// this potentially means that some of next msgs will fail to load
			// to avoid that requery
			if delmodids.contain(mod_id) && i+1 < len(posts) {
				sp.log.LogPrintf(
					DEBUG, "setmodpriv: delmodid %d is ours, requerying", mod_id)
				// msg we just deleted was made by mod we just upp'd
				// that means that it may be msg in query we just made
				// it's unsafe to proceed with current cached query

				numProcessed += i + 1
				numRequest = maxInt(
					len(posts)-i-1,
					minInt(
						fetchMsgsOnce,
						fetchMsgsTotal-numProcessed))

				continue requery
			}
		}

		numProcessed += len(posts)

		if len(posts) < numRequest {
			// returned less which means that this is the end
			// mark the fact that this is the end
			t_date_sent.Valid = false
			t_g_p_id.Valid = false
			t_b_id.Valid = false
			// and break out
			break
		}

		if numProcessed >= fetchMsgsTotal {
			break
		}

		numRequest = minInt(fetchMsgsOnce, fetchMsgsTotal-numProcessed)
	}

	// now put out job state
	if t_g_p_id.Valid {
		_, err = mcs.Exec(j_id, t_date_sent, t_g_p_id, t_b_id)
		if err != nil {
			err = sp.SQLError("job set", err)
			return
		}
	} else {
		_, err = mcd.Exec(j_id)
		if err != nil {
			err = sp.SQLError("job del", err)
			return
		}
	}

	// done
	return
}

// reads jobs n shit

func (sp *PSQLIB) modset_jobprocessor(notif <-chan struct{}) {
	doscan := true
	for {
		if !doscan {
			// wait
			_, ok := <-notif
			if !ok {
				return
			}
		}
		///
	}
}
