package pipostbase

import (
	"database/sql"
	"encoding/hex"
	"strings"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"

	. "nksrv/lib/logx"
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

func mustUnmarshal(x interface{}, j xtypes.JSONText) {
	err := j.Unmarshal(&x)
	if err != nil {
		panic("json unmarshal")
	}
}

type modPrivFetch struct {
	m_g_cap   sql.NullString
	m_b_cap   map[string]string
	m_b_cap_j xtypes.JSONText

	m_g_caplvl   []sql.NullInt32
	m_b_caplvl   map[string]string
	m_b_caplvl_j xtypes.JSONText

	mi_g_cap   sql.NullString
	mi_b_cap   map[string]string
	mi_b_cap_j xtypes.JSONText

	mi_g_caplvl   []sql.NullInt32
	mi_b_caplvl   map[string]string
	mi_b_caplvl_j xtypes.JSONText
}

func (f *modPrivFetch) unmarshalJSON() {
	mustUnmarshal(&f.m_b_cap, f.m_b_cap_j)
	mustUnmarshal(&f.m_b_caplvl, f.m_b_caplvl_j)
	mustUnmarshal(&f.mi_b_cap, f.mi_b_cap_j)
	mustUnmarshal(&f.mi_b_caplvl, f.mi_b_caplvl_j)
}

func (f *modPrivFetch) parse() (mcc ModCombinedCaps) {
	if f.m_g_cap.Valid {
		mcc.ModCap.Cap = StrToCap(f.m_g_cap.String)
	}
	if f.m_g_caplvl != nil {
		mcc.ModCap = processCapLevel(mcc.ModCap, f.m_g_caplvl)
	}

	if f.mi_g_cap.Valid {
		mcc.ModInheritCap.Cap = StrToCap(f.mi_g_cap.String)
	}
	if f.mi_g_caplvl != nil {
		mcc.ModInheritCap = processCapLevel(mcc.ModInheritCap, f.mi_g_caplvl)
	}

	mcc.ModBoardCap = make(ModBoardCap)
	mcc.ModBoardCap.TakeIn(f.m_b_cap, f.m_b_caplvl)

	mcc.ModInheritBoardCap = make(ModBoardCap)
	mcc.ModInheritBoardCap.TakeIn(f.mi_b_cap, f.mi_b_caplvl)

	return
}

func (sp *PSQLIB) registeredMod(tx *sql.Tx, pubkeystr string) (rmi regModInfo, err error) {

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

		var f modPrivFetch

		err = st.QueryRow(pubkeystr).Scan(
			&rmi.modid,

			&f.m_g_cap,
			&f.m_b_cap_j,
			pq.Array(&f.m_g_caplvl),
			&f.m_b_caplvl_j,

			&f.mi_g_cap,
			&f.mi_b_cap_j,
			pq.Array(&f.mi_g_caplvl),
			&f.mi_b_caplvl_j)

		if err != nil {

			if err == sql.ErrNoRows && x < 100 {

				x++

				sp.log.LogPrintf(DEBUG, "REGMOD %s retry", pubkeystr)

				continue
			}

			err = sp.sqlError("st_web_autoregister_mod queryrowscan", err)
			return
		}

		f.unmarshalJSON()

		// enough to check only usable flags
		rmi.actionable = f.m_g_cap.Valid || len(f.m_b_cap) != 0 ||
			f.m_g_caplvl != nil || len(f.m_b_caplvl) != 0

		rmi.ModCombinedCaps = f.parse()

		return
	}
}

func makeCapLvlArray(mc ModCap) interface{} {
	var x [caplvlx_num]sql.NullInt32
	for i := range x {
		x[i].Int32 = int32(mc.CapLevel[i])
		x[i].Valid = mc.CapLevel[i] >= 0
	}
	return pq.Array(x)
}

func (sp *PSQLIB) setModCap(
	tx *sql.Tx, pubkeystr, group string, m_cap, mi_cap ModCap) (err error) {

	// do key update
	var dummy int32
	// this probably should lock relevant row.
	// that should block reads of this row I think?
	// which would mean no further new mod posts for this key
	var r *sql.Row

	m_caplvl := makeCapLvlArray(m_cap)
	mi_caplvl := makeCapLvlArray(mi_cap)

	if group == "" {

		ust := tx.Stmt(sp.st_prep[st_mod_set_mod_priv])

		r = ust.QueryRow(

			pubkeystr,

			m_cap.Cap.String(),
			m_caplvl,

			mi_cap.Cap.String(),
			mi_caplvl)

	} else {

		ust := tx.Stmt(sp.st_prep[st_mod_set_mod_priv_group])

		r = ust.QueryRow(

			pubkeystr,
			group,

			m_cap.Cap.String(),
			m_caplvl,

			mi_cap.Cap.String(),
			mi_caplvl)

	}

	err = r.Scan(&dummy)

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
		"setmodpriv: %s priv changed", pubkeystr)

	return
}

func (sp *PSQLIB) DemoSetModCap(
	mods []string, group string, modCap, modInheritCap ModCap) {

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

	// inheritable priv implies usable priv
	modCap = modCap.Merge(modInheritCap)

	for _, s := range mods {
		sp.log.LogPrintf(INFO, "setmodpriv %s %s", s, modCap.String())

		err = sp.setModCap(tx, s, group, modCap, modInheritCap)
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

func (sp *PSQLIB) checkFiles() {
	//
	//sp.st_prep[st_mod_load_files].
}

type phdata struct {
	ph_ban     bool
	ph_banpriv caplvl_type
}

type articlecheckinfo struct {
	g_p_id   uint64
	has_real bool
	has_ph   bool
	phdata
}

type savephdata struct {
	ph_ban     sql.NullBool
	ph_banpriv sql.NullInt32
}

func (sp *PSQLIB) checkArticleForPush(
	cmsgids TCoreMsgIDStr) (i articlecheckinfo, e error) {

	st := sp.st_prep[st_mod_check_article_for_push]

	var banpriv sql.NullInt32
	e = st.QueryRow(cmsgids).
		Scan(
			&i.g_p_id,
			&i.has_real,
			&i.has_ph,
			&i.ph_ban,
			&banpriv)

	if e != nil {
		if e == sql.ErrNoRows {
			e = nil
			return
		}
		e = sp.sqlError("", e)
		return
	}

	if !i.has_real && !i.has_ph {
		panic("empty")
	}

	if banpriv.Valid {
		if uint32(banpriv.Int32) > caplvl_maxval {
			panic("too big")
		}
		i.ph_banpriv = caplvl_type(banpriv.Int32)
	} else {
		i.ph_banpriv = -1
	}

	return
}

func (sp *PSQLIB) deletePHForPush(
	g_p_id uint64, phd phdata) (ok bool, sphd savephdata, e error) {

	st := sp.st_prep[st_mod_delete_ph_for_push]

	banpriv := sql.NullInt32{
		Valid: phd.ph_banpriv >= 0,
		Int32: int32(phd.ph_banpriv),
	}
	e = st.
		QueryRow(
			g_p_id,
			phd.ph_ban,
			banpriv).
		Scan(
			&sphd.ph_ban,
			&sphd.ph_banpriv)
	if e != nil {
		if e == sql.ErrNoRows {
			e = nil
			return
		}
		e = sp.sqlError("", e)
		return
	}
	ok = true
	return
}

func (sp *PSQLIB) addPHAfterPush(g_p_id uint64, sphd savephdata) (e error) {

	st := sp.st_prep[st_mod_add_ph_after_push]

	_, e = st.
		Exec(
			g_p_id,
			sphd.ph_ban,
			sphd.ph_banpriv)
	if e != nil {
		e = sp.sqlError("", e)
	}
	return
}
