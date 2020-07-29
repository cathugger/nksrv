package pipostbase

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"

	. "nksrv/lib/logx"
	"nksrv/lib/psqlib/internal/pibase"
	"nksrv/lib/psqlib/internal/pibasemod"
	"nksrv/lib/psqlib/internal/pibasenntp"
	"nksrv/lib/thumbnailer"
)

func PickThumbPlan(sp *pibase.PSQLIB, isReply, isSage bool) thumbnailer.ThumbPlan {
	if isReply {
		if !isSage {
			return sp.ThmPlanForPost
		} else {
			return sp.ThmPlanForSage
		}
	} else {
		return sp.ThmPlanForOP
	}
}

func MustMarshal(x interface{}) []byte {
	s, err := json.Marshal(x)
	if err != nil {
		panic("JSON Marshal: " + err.Error())
	}
	return s
}

func MustUnmarshal(x interface{}, j xtypes.JSONText) {
	err := j.Unmarshal(&x)
	if err != nil {
		panic("JSON Unmarshal: " + err.Error())
	}
}

type ModPrivFetch struct {
	ModGlobalCap    sql.NullString
	ModBoardCap     map[string]string
	ModBoardCapJSON xtypes.JSONText

	ModGlobalCapLvl    []sql.NullInt32
	ModBoardCapLvl     map[string]string
	ModBoardCapLvlJSON xtypes.JSONText

	ModIGlobalCap    sql.NullString
	ModIBoardCap     map[string]string
	ModIBoardCapJSON xtypes.JSONText

	ModIGlobalCapLvl    []sql.NullInt32
	ModIBoardCapLvl     map[string]string
	ModIBoardCapLvlJSON xtypes.JSONText
}

func (f *ModPrivFetch) unmarshalJSON() {
	MustUnmarshal(&f.ModBoardCap, f.ModBoardCapJSON)
	MustUnmarshal(&f.ModBoardCapLvl, f.ModBoardCapLvlJSON)
	MustUnmarshal(&f.ModIBoardCap, f.ModIBoardCapJSON)
	MustUnmarshal(&f.ModIBoardCapLvl, f.ModIBoardCapLvlJSON)
}

func (f *ModPrivFetch) parse() (mcc pibasemod.ModCombinedCaps) {
	if f.ModGlobalCap.Valid {
		mcc.ModCap.Cap = pibasemod.StrToCap(f.ModGlobalCap.String)
	}
	if f.ModGlobalCapLvl != nil {
		mcc.ModCap = pibasemod.ProcessCapLevel(mcc.ModCap, f.ModGlobalCapLvl)
	}

	if f.ModIGlobalCap.Valid {
		mcc.ModInheritCap.Cap = pibasemod.StrToCap(f.ModIGlobalCap.String)
	}
	if f.ModIGlobalCapLvl != nil {
		mcc.ModInheritCap = pibasemod.ProcessCapLevel(mcc.ModInheritCap, f.ModIGlobalCapLvl)
	}

	mcc.ModBoardCap = make(pibasemod.ModBoardCap)
	mcc.ModBoardCap.TakeIn(f.ModBoardCap, f.ModBoardCapLvl)

	mcc.ModInheritBoardCap = make(pibasemod.ModBoardCap)
	mcc.ModInheritBoardCap.TakeIn(f.ModIBoardCap, f.ModIBoardCapLvl)

	return
}

func registeredMod(sp *pibase.PSQLIB, tx *sql.Tx, pubkeystr string) (rmi regModInfo, err error) {

	// mod posts MAY later come back and want more of things in this table (if they eval/GC modposts)
	// at which point we're fucked because moddel posts also will exclusively block files table
	// and then we won't be able to insert into it..
	_, err = tx.Exec("LOCK ib0.modlist IN EXCLUSIVE MODE")
	if err != nil {
		err = sp.SQLError("lock ib0.modlist query", err)
		return
	}

	sp.Log.LogPrintf(DEBUG, "REGMOD %s done locking ib0.modlist", pubkeystr)

	st := tx.Stmt(sp.StPrep[pibase.St_mod_autoregister_mod])
	x := 0
	for {

		var f ModPrivFetch

		err = st.QueryRow(pubkeystr).Scan(
			&rmi.modid,

			&f.ModGlobalCap,
			&f.ModBoardCapJSON,
			pq.Array(&f.ModGlobalCapLvl),
			&f.ModBoardCapLvlJSON,

			&f.ModIGlobalCap,
			&f.ModIBoardCapJSON,
			pq.Array(&f.ModIGlobalCapLvl),
			&f.ModIBoardCapLvlJSON)

		if err != nil {

			if err == sql.ErrNoRows && x < 100 {

				x++

				sp.Log.LogPrintf(DEBUG, "REGMOD %s retry", pubkeystr)

				continue
			}

			err = sp.SQLError("st_web_autoregister_mod queryrowscan", err)
			return
		}

		f.unmarshalJSON()

		// enough to check only usable flags
		rmi.actionable = f.ModGlobalCap.Valid || len(f.ModBoardCap) != 0 ||
			f.ModGlobalCapLvl != nil || len(f.ModBoardCapLvl) != 0

		rmi.ModCombinedCaps = f.parse()

		return
	}
}

func makeCapLvlArray(mc pibasemod.ModCap) interface{} {
	var x [pibasemod.CapLvlX_Num]sql.NullInt32
	for i := range x {
		x[i].Int32 = int32(mc.CapLevel[i])
		x[i].Valid = mc.CapLevel[i] >= 0
	}
	return pq.Array(x)
}

func setModCap(
	sp *pibase.PSQLIB, tx *sql.Tx,
	pubkeystr, group string, m_cap, mi_cap pibasemod.ModCap) (err error) {

	// do key update
	var dummy int32
	// this probably should lock relevant row.
	// that should block reads of this row I think?
	// which would mean no further new mod posts for this key
	var r *sql.Row

	m_caplvl := makeCapLvlArray(m_cap)
	mi_caplvl := makeCapLvlArray(mi_cap)

	if group == "" {

		ust := tx.Stmt(sp.StPrep[pibase.St_mod_set_mod_priv])

		r = ust.QueryRow(

			pubkeystr,

			m_cap.Cap.String(),
			m_caplvl,

			mi_cap.Cap.String(),
			mi_caplvl)

	} else {

		ust := tx.Stmt(sp.StPrep[pibase.St_mod_set_mod_priv_group])

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
			sp.Log.LogPrintf(DEBUG, "setmodpriv: %s priv unchanged", pubkeystr)
			err = nil
			return
		}
		err = sp.SQLError("st_web_set_mod_priv queryrowscan", err)
		return
	}

	sp.Log.LogPrintf(DEBUG,
		"setmodpriv: %s priv changed", pubkeystr)

	return
}

func DemoSetModCap(
	sp *pibase.PSQLIB,
	mods []string, group string, modCap, modInheritCap pibasemod.ModCap) {

	var err error

	for i, s := range mods {
		if _, err = hex.DecodeString(s); err != nil {
			sp.Log.LogPrintf(ERROR, "invalid modid %q", s)
			return
		}
		// we use uppercase (I forgot why)
		mods[i] = strings.ToUpper(s)
	}

	tx, err := sp.DB.DB.Begin()
	if err != nil {
		err = sp.SQLError("tx begin", err)
		sp.Log.LogPrintf(ERROR, "%v", err)
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
		sp.Log.LogPrintf(INFO, "setmodpriv %s %s", s, modCap.String())

		err = setModCap(sp, tx, s, group, modCap, modInheritCap)
		if err != nil {
			sp.Log.LogPrintf(ERROR, "%v", err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = sp.SQLError("tx commit", err)
		sp.Log.LogPrintf(ERROR, "%v", err)
		return
	}
}

func checkFiles(sp *pibase.PSQLIB) {
	//
	//sp.st_prep[st_mod_load_files].
}

type phdata struct {
	ph_ban     bool
	ph_banpriv pibasemod.TCapLvl
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

func checkArticleForPush(
	sp *pibase.PSQLIB,
	cmsgids pibasenntp.TCoreMsgIDStr) (i articlecheckinfo, e error) {

	st := sp.StPrep[pibase.St_mod_check_article_for_push]

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
		e = sp.SQLError("", e)
		return
	}

	if !i.has_real && !i.has_ph {
		panic("empty")
	}

	if banpriv.Valid {
		if uint32(banpriv.Int32) > pibasemod.CapLvl_MaxVal {
			panic("too big")
		}
		i.ph_banpriv = pibasemod.TCapLvl(banpriv.Int32)
	} else {
		i.ph_banpriv = -1
	}

	return
}

func deletePHForPush(
	sp *pibase.PSQLIB,
	g_p_id uint64, phd phdata) (ok bool, sphd savephdata, e error) {

	st := sp.StPrep[pibase.St_mod_delete_ph_for_push]

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
		e = sp.SQLError("", e)
		return
	}
	ok = true
	return
}

func addPHAfterPush(sp *pibase.PSQLIB, g_p_id uint64, sphd savephdata) (e error) {

	st := sp.StPrep[pibase.St_mod_add_ph_after_push]

	_, e = st.
		Exec(
			g_p_id,
			sphd.ph_ban,
			sphd.ph_banpriv)
	if e != nil {
		e = sp.SQLError("", e)
	}
	return
}
