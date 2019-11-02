package psqlib

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/lib/pq"

	"nksrv/lib/altthumber"
	"nksrv/lib/demoib"
	"nksrv/lib/emime"
	fl "nksrv/lib/filelogger"
	"nksrv/lib/fstore"
	"nksrv/lib/logx"
	"nksrv/lib/psql"
	"nksrv/lib/psql/testutil"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/thumbnailer/gothm"
)

var cfgAltThm = altthumber.AltThumber(demoib.DemoAltThumber{})

var cfgPSQLIB = Config{
	NodeName:   "nekochan",
	SrcCfg:     &fstore.Config{"_demo/demoib0/src"},
	ThmCfg:     &fstore.Config{"_demo/demoib0/thm"},
	NNTPFSCfg:  &fstore.Config{"_demo/demoib0/nntp"},
	AltThumber: &cfgAltThm,
	TBuilder:   gothm.DefaultConfig,
	TCfgThread: &thumbnailer.ThumbConfig{
		Width:       250,
		Height:      250,
		AudioWidth:  350,
		AudioHeight: 350,
		Color:       "#EEF2FF",
	},
	TCfgReply: &thumbnailer.ThumbConfig{
		Width:       200,
		Height:      200,
		AudioWidth:  350,
		AudioHeight: 350,
		Color:       "#D6DAF0",
	},
}

func newLogger() (lgr logx.LoggerX) {
	lgr, err := fl.NewFileLogger(os.Stderr, logx.DEBUG, fl.ColorAuto)
	if err != nil {
		panic("fl.NewFileLogger err: " + err.Error())
	}
	return
}

func panicErr(err error, str string) {
	if err != nil {
		panic(str + ": " + err.Error())
	}
}

func init() {
	err := os.Chdir("../../../..")
	panicErr(err, "chdir failed")
	ok, err := emime.LoadMIMEDatabase("mime.types")
	panicErr(err, "can't load mime")
	if !ok {
		panic("didn't find mime.types")
	}
}

func TestInit(t *testing.T) {
	dbn := testutil.MakeTestDB()
	defer testutil.DropTestDB(dbn)

	lgr := newLogger()

	db, err := psql.OpenAndPrepare(psql.Config{
		ConnStr: "user=" + testutil.TestUser +
			" dbname=" + dbn +
			" host=" + testutil.PSQLHost,
		Logger: lgr,
	})
	panicErr(err, "OAP err")

	defer func() {
		err = db.Close()
		panicErr(err, "db close err")
	}()

	psqlibcfg := cfgPSQLIB
	psqlibcfg.DB = &db
	psqlibcfg.Logger = &lgr

	dbib, err := NewInitAndPrepare(psqlibcfg)
	panicErr(err, "NewInitAndPrepare err")

	err = dbib.Close()
	panicErr(err, "dbib close err")
}

func TestCalcPriv(t *testing.T) {
	dbn := testutil.MakeTestDB()
	defer testutil.DropTestDB(dbn)

	lgr := newLogger()

	db, err := psql.OpenAndPrepare(psql.Config{
		ConnStr: "user=" + testutil.TestUser +
			" dbname=" + dbn +
			" host=" + testutil.PSQLHost,
		Logger: lgr,
	})
	panicErr(err, "OAP err")

	defer func() {
		err = db.Close()
		panicErr(err, "db close err")
	}()

	psqlibcfg := cfgPSQLIB
	psqlibcfg.DB = &db
	psqlibcfg.Logger = &lgr

	dbib, err := NewInitAndPrepare(psqlibcfg)
	panicErr(err, "NewInitAndPrepare err")

	defer func() {
		err = dbib.Close()
		panicErr(err, "dbib close err")
	}()

	tx, err := db.DB.Begin()
	panicErr(err, "db.DB.Begin err")

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	lvl_none := [caplvlx_num]int16{-1}
	lvl_one := [caplvlx_num]int16{0}
	capsets := [...]struct {
		Key    string
		Group  string
		ModCap ModCap
	}{
		{Key: "0", ModCap: ModCap{CapLevel: lvl_none}},
		{Key: "0", ModCap: ModCap{CapLevel: lvl_none}},
		{Key: "0", ModCap: ModCap{CapLevel: lvl_one}},
		{Key: "0", ModCap: ModCap{CapLevel: lvl_none}},
		{Key: "0", ModCap: ModCap{CapLevel: lvl_one}},

		{Key: "1", ModCap: ModCap{CapLevel: lvl_one}},
		{Key: "1", ModCap: ModCap{CapLevel: lvl_one}},
		{Key: "1", ModCap: ModCap{CapLevel: lvl_none}},

		{Key: "2", ModCap: ModCap{Cap: cap_delpost}},

		{Key: "3", Group: "test", ModCap: ModCap{Cap: cap_delpost, CapLevel: lvl_none}},
		{Key: "3", Group: "test", ModCap: ModCap{Cap: cap_delpost, CapLevel: lvl_none}},
		{Key: "3", Group: "test", ModCap: ModCap{Cap: cap_delpost, CapLevel: lvl_one}},

		{Key: "4", Group: "test", ModCap: ModCap{Cap: cap_delpost, CapLevel: lvl_one}},
		{Key: "4", Group: "test", ModCap: ModCap{Cap: cap_delpost, CapLevel: lvl_none}},
		{Key: "4", Group: "test", ModCap: ModCap{Cap: cap_delpost, CapLevel: lvl_none}},

		{Key: "5", ModCap: ModCap{Cap: cap_delpost, CapLevel: lvl_one}},
		{Key: "5", Group: "test", ModCap: ModCap{Cap: cap_delpost, CapLevel: lvl_one}},
	}
	for i, cs := range capsets {
		err = dbib.setModCap(tx, cs.Key, cs.Group, cs.ModCap)
		panicErr(err, fmt.Sprintf("capset %d", i))
	}

	err = tx.Commit()
	panicErr(err, "tx.Commit err")

	q := `
SELECT
	mod_pubkey,
	automanage,
	mod_cap,
	mod_bcap,
	mod_caplvl,
	mod_bcaplvl
FROM
	ib0.modlist
ORDER BY
	mod_pubkey
`
	rows, err := db.DB.Query(q)
	panicErr(err, "db.DB.Query err")
	i := 0
	type res_t struct {
		PubKey     string
		AutoManage bool
		ModCap     sql.NullString
		ModBCap    sql.NullString
		ModCapLvl  sql.NullString
		ModBCapLvl sql.NullString
	}
	nullcap := sql.NullString{String: "000000000000", Valid: true}
	zeropriv := sql.NullString{String: "{0}", Valid: true}
	nullpriv := sql.NullString{String: "{NULL}", Valid: true}
	expres := [...]res_t{
		{PubKey: "0", ModCap: nullcap, ModCapLvl: zeropriv},
		{PubKey: "1", ModCap: nullcap, ModCapLvl: nullpriv},
		{
			PubKey:    "2",
			ModCap:    sql.NullString{String: "010000000000", Valid: true},
			ModCapLvl: zeropriv,
		},
		{
			PubKey:     "3",
			ModBCap:    sql.NullString{String: "{\"test\": \"010000000000\"}", Valid: true},
			ModBCapLvl: sql.NullString{String: "{\"test\": \"{0}\"}", Valid: true},
		},
		{
			PubKey:     "4",
			ModBCap:    sql.NullString{String: "{\"test\": \"010000000000\"}", Valid: true},
			ModBCapLvl: sql.NullString{String: "{\"test\": \"{NULL}\"}", Valid: true},
		},
		{
			PubKey:     "5",
			ModCap:     sql.NullString{String: "010000000000", Valid: true},
			ModCapLvl:  zeropriv,
			ModBCap:    sql.NullString{String: "{\"test\": \"010000000000\"}", Valid: true},
			ModBCapLvl: sql.NullString{String: "{\"test\": \"{0}\"}", Valid: true},
		},
	}
	for rows.Next() {
		var x res_t
		err = rows.Scan(
			&x.PubKey, &x.AutoManage,
			&x.ModCap, &x.ModBCap, &x.ModCapLvl, &x.ModBCapLvl)
		panicErr(err, "rows.Scan err")
		if i >= len(expres) {
			t.Errorf("res: too many rows: %#v", x)
			continue
		}
		if x != expres[i] {
			t.Errorf("res: %d not equal, got: %#v", i, x)
		}
		i++
	}
	if i != len(expres) {
		t.Errorf("res: too little rows")
	}

	// check if changes list properly reflect changes
	type cl_t struct {
		j_id        int64
		mod_id      int64
		t_date_sent pq.NullTime
		t_g_p_id    sql.NullInt64
		t_b_id      sql.NullInt32
	}
	var expcl = [...]cl_t{
		{j_id: 1, mod_id: 1},
		{j_id: 4, mod_id: 2},
		{j_id: 5, mod_id: 4},
		{j_id: 6, mod_id: 5},
		{j_id: 7, mod_id: 6},
	}
	checkexp := func(i int) {
		tx, err := db.DB.Begin()
		panicErr(err, "db.DB.Begin err")
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()
		cmt := func() {
			err = tx.Commit()
			panicErr(err, "cl tx.Commit err")
		}

		var x cl_t

		err = dbib.
			st_prep[st_mod_joblist_modlist_changes_get].
			QueryRow().
			Scan(&x.j_id, &x.mod_id, &x.t_date_sent, &x.t_g_p_id, &x.t_b_id)
		if err != nil {
			if err == sql.ErrNoRows {
				if i < len(expcl) {
					// it shouldn't have ended yet
					t.Errorf("cl: too little rows")
				}
				cmt()
				return
			}
			panicErr(err, "cl queryrowscan err")
		}

		if i >= len(expcl) {
			t.Errorf("cl: too many rows: %#v", x)
		} else if x != expcl[i] {
			t.Errorf("cl: %d not equal, got: %#v", i, x)
		}

		_, err = dbib.st_prep[st_mod_joblist_modlist_changes_del].Exec(x.j_id)
		panicErr(err, "cl del exec err")

		cmt()
	}

	for i := range expcl {
		checkexp(i)
	}
	checkexp(i)
}
