package pibase

// psql imageboard module

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/crypto/ed25519"

	"nksrv/lib/altthumber"
	"nksrv/lib/cacheengine"
	"nksrv/lib/fstore"
	. "nksrv/lib/logx"
	"nksrv/lib/mail/form"
	"nksrv/lib/psql"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/webcaptcha"
)

type PSQLIB struct {
	db  psql.PSQL
	log LogToX

	src    fstore.FStore
	thm    fstore.FStore
	nntpfs fstore.FStore

	pending2src fstore.Mover
	pending2thm fstore.Mover

	nntpce cacheengine.CacheEngine

	thumbnailer  thumbnailer.Thumbnailer
	tplan_thread thumbnailer.ThumbPlan
	tplan_reply  thumbnailer.ThumbPlan
	tplan_sage   thumbnailer.ThumbPlan
	altthumb     altthumber.AltThumber

	ffo                formFileOpener
	fpp                form.ParserParams
	textPostParamFunc  func(string) bool
	instance           string
	maxArticleBodySize int64
	webcaptcha         *webcaptcha.WebCaptcha
	webFrontendKey     ed25519.PrivateKey

	ngp_global    newGroupPolicy
	ngp_anypuller newGroupPolicy
	ngp_anyserver newGroupPolicy

	st_prep [st_max]*sql.Stmt

	// newthread prepared statements and locking
	ntStmts map[int]*sql.Stmt
	ntMutex sync.RWMutex

	// newpost prepared statements and locking
	npStmts map[npTuple]*sql.Stmt
	npMutex sync.RWMutex

	puller_nonce int64

	noFileSync bool
}


// readonly for now


func (sp *PSQLIB) Prepare() (err error) {
	err = sp.prepareStatements()
	if err != nil {
		return
	}

	return
}

func (sp *PSQLIB) Close() error {
	return sp.closeStatements()
}

func (dbib *PSQLIB) InitAndPrepare() (err error) {
	valid, err := dbib.CheckDB()
	if err != nil {
		return fmt.Errorf("error checking: %v", err)
	}
	if !valid {
		dbib.log.LogPrint(NOTICE,
			"uninitialized db, attempting to initialize")

		err = dbib.InitDB()
		if err != nil {
			return fmt.Errorf("error initializing: %v", err)
		}

		valid, err = dbib.CheckDB()
		if err != nil {
			return fmt.Errorf("error checking (2): %v", err)
		}
		if !valid {
			return errors.New("database still not valid after initialization")
		}
	}

	err = dbib.Prepare()
	if err != nil {
		return
	}

	return
}

func NewInitAndPrepare(cfg Config) (db *PSQLIB, err error) {
	db, err = NewPSQLIB(cfg)
	if err != nil {
		return
	}

	err = db.InitAndPrepare()
	if err != nil {
		return
	}

	return
}
