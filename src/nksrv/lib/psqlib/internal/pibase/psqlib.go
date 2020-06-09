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

	"nksrv/lib/psqlib/internal/pigpolicy"
)

type PSQLIB struct {
	// database handle
	DB psql.PSQL
	// this psqlib instance logger
	Log LogToX
	// file storage stuff
	Src    fstore.FStore
	Thm    fstore.FStore
	NNTPFS fstore.FStore
	// movers for posting
	PendingToSrc fstore.Mover
	PendingToThm fstore.Mover
	// for caching of generated NetNews articles
	NNTPCE cacheengine.CacheEngine
	// thumbnailing things
	Thumbnailer      thumbnailer.Thumbnailer
	ThmPlanForThread thumbnailer.ThumbPlan
	ThmPlanForReply  thumbnailer.ThumbPlan
	ThmPlanForSage   thumbnailer.ThumbPlan
	AltThumber       altthumber.AltThumber
	// HTTP POST form handling
	FFO               formFileOpener
	FPP               form.ParserParams
	TextPostParamFunc func(string) bool

	Instance           string
	MaxArticleBodySize int64
	WebCaptcha         *webcaptcha.WebCaptcha
	WebFrontendKey     ed25519.PrivateKey

	NGPGlobal    pigpolicy.NewGroupPolicy
	NGPAnyPuller pigpolicy.NewGroupPolicy
	NGPAnyServer pigpolicy.NewGroupPolicy

	StPrep [StMax]*sql.Stmt

	// newthread prepared statements and locking
	NTStmts map[int]*sql.Stmt
	NTMutex sync.RWMutex

	// newpost prepared statements and locking
	NPStmts map[NPTuple]*sql.Stmt
	NPMutex sync.RWMutex

	PullerNonce int64

	NoFileSync bool
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
