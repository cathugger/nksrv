package psqlib

import (
	"os"
	"testing"

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

func init() {
	err := os.Chdir("../../../..")
	if err != nil {
		panic("chdir failed: " + err.Error())
	}
	ok, err := emime.LoadMIMEDatabase("mime.types")
	if err != nil {
		panic("can't load mime: " + err.Error())
	}
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
	if err != nil {
		panic("OAP err: " + err.Error())
	}

	defer func() {
		err = db.Close()
		if err != nil {
			panic("db close err: " + err.Error())
		}
	}()

	psqlibcfg := cfgPSQLIB
	psqlibcfg.DB = &db
	psqlibcfg.Logger = &lgr

	dbib, err := NewInitAndPrepare(psqlibcfg)
	if err != nil {
		panic("NewInitAndPrepare err: " + err.Error())
	}

	err = dbib.Close()
	if err != nil {
		panic("dbib close err: " + err.Error())
	}
}
