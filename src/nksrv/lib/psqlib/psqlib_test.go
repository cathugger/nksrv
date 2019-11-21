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

var lvl_none = [caplvlx_num]int16{-1}
var lvl_one = [caplvlx_num]int16{0}

func setModCaps1tx(dbib *PSQLIB, tx *sql.Tx) {
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
		err := dbib.setModCap(tx, cs.Key, cs.Group, cs.ModCap, noneModCap)
		panicErr(err, fmt.Sprintf("capset %d", i))
	}
}

func setModCaps1(dbib *PSQLIB) {
	tx, err := dbib.db.DB.Begin()
	panicErr(err, "db.DB.Begin err")

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	setModCaps1tx(dbib, tx)

	err = tx.Commit()
	panicErr(err, "tx.Commit err")
}

func validateModCaps1(t *testing.T, dbib *PSQLIB) {
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
	rows, err := dbib.db.DB.Query(q)
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
}

func validateChangeList1(t *testing.T, dbib *PSQLIB) {
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
		tx, err := dbib.db.DB.Begin()
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
		var f modPrivFetch

		err = dbib.
			st_prep[st_mod_joblist_modlist_changes_get].
			QueryRow().
			Scan(
				&x.j_id,
				&x.mod_id,

				&x.t_date_sent,
				&x.t_g_p_id,
				&x.t_b_id,

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

	i := 0
	for ; i < len(expcl); i++ {
		checkexp(i)
	}
	checkexp(i)
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

	setModCaps1(dbib)

	validateModCaps1(t, dbib)

	validateChangeList1(t, dbib)
}

func submitFromFile(dbib *PSQLIB, name string) (error, bool) {

	f, e := os.Open("testdata/msg/" + name + ".eml")
	panicErr(e, "open")
	defer f.Close()

	return dbib.netnewsHandleSubmissionDirectly(f, false)
}

func insertFiles1(t *testing.T, dbib *PSQLIB) {
	tests := [...]struct {
		name          string
		shouldsucceed bool
	}{
		{"msg1", true},
		{"msg2", true},
		{"msg3", false},

		{"mod1", true},
		{"mod2", true},
		{"mod3", true},
	}
	for i := range tests {
		ee, unexp := submitFromFile(dbib, tests[i].name)
		if ee != nil {
			if unexp {
				t.Errorf("! unexpected submission err: %v", ee)
			} else if tests[i].shouldsucceed {
				t.Errorf("! submission error when should succeed, err: %v", ee)
			} else {
				t.Logf("+ submission error when should error, err: %v", ee)
			}
		} else {
			if !tests[i].shouldsucceed {
				t.Errorf("! submission succeed when should error")
			} else {
				t.Logf("+ submission succeed when should succeed")
			}
		}
	}
}

func TestPost(t *testing.T) {

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
	psqlibcfg.NGPGlobal = "*"

	dbib, err := NewInitAndPrepare(psqlibcfg)
	panicErr(err, "NewInitAndPrepare err")

	defer func() {
		err = dbib.Close()
		panicErr(err, "dbib close err")
	}()

	insertFiles1(t, dbib)

	dbib.DemoSetModCap(
		[]string{"2d2ca0ed8361b5569786e41b8fd7a39de8fc064270966b57510b0c7a8d1a7215"},
		"",
		ModCap{Cap: cap_delpost, CapLevel: lvl_none},
		noneModCap)

	n_proc := 0
	for {
		hadw, e := dbib.modset_processJobOnce(1, 1)
		panicErr(e, "modset_processJobOnce")
		if !hadw {
			break
		}
		n_proc++
	}

	if n_proc != 4 {
		t.Errorf("! n_proc doesn't match got: %v", n_proc)
	} else {
		t.Logf("+ n_proc matches")
	}
}

func TestPost2(t *testing.T) {
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
	psqlibcfg.NGPGlobal = "*"

	dbib, err := NewInitAndPrepare(psqlibcfg)
	panicErr(err, "NewInitAndPrepare err")

	defer func() {
		err = dbib.Close()
		panicErr(err, "dbib close err")
	}()

	dbib.DemoSetModCap(
		[]string{"2d2ca0ed8361b5569786e41b8fd7a39de8fc064270966b57510b0c7a8d1a7215"},
		"",
		ModCap{Cap: cap_delpost, CapLevel: lvl_none},
		noneModCap)

	insertFiles1(t, dbib)

	n_proc := 0
	for {
		hadw, e := dbib.modset_processJobOnce(1, 1)
		panicErr(e, "modset_processJobOnce")
		if !hadw {
			break
		}
		n_proc++
	}

	if n_proc != 0 {
		t.Errorf("! n_proc doesn't match got: %v", n_proc)
	} else {
		t.Logf("+ n_proc matches")
	}
}
