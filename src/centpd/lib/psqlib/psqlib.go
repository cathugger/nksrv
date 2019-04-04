package psqlib

// psql imageboard module

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"centpd/lib/altthumber"
	"centpd/lib/fstore"
	. "centpd/lib/logx"
	"centpd/lib/mail/form"
	"centpd/lib/nilthumbnailer"
	"centpd/lib/psql"
	"centpd/lib/thumbnailer"
)

type PSQLIB struct {
	db                   psql.PSQL
	log                  Logger
	src                  fstore.FStore
	thm                  fstore.FStore
	nntpfs               fstore.FStore
	nntpmgr              nntpCacheMgr
	thumbnailer          thumbnailer.Thumbnailer
	tplan_thread         thumbnailer.ThumbPlan
	tplan_reply          thumbnailer.ThumbPlan
	tplan_sage           thumbnailer.ThumbPlan
	altthumb             altthumber.AltThumber
	ffo                  formFileOpener
	fpp                  form.ParserParams
	instance             string
	maxArticleBodySize   int64
	autoAddNNTPPostGroup bool

	st_prep [st_max]*sql.Stmt

	// newthread prepared statements and locking
	ntStmts map[int]*sql.Stmt
	ntMutex sync.RWMutex

	// newpost prepared statements and locking
	npStmts map[npTuple]*sql.Stmt
	npMutex sync.RWMutex

	puller_nonce int64
}

type Config struct {
	DB                 *psql.PSQL
	Logger             *LoggerX
	SrcCfg             *fstore.Config
	ThmCfg             *fstore.Config
	NNTPFSCfg          *fstore.Config
	TBuilder           thumbnailer.ThumbnailerBuilder
	TCfgThread         *thumbnailer.ThumbConfig
	TCfgReply          *thumbnailer.ThumbConfig
	TCfgSage           *thumbnailer.ThumbConfig
	AltThumber         *altthumber.AltThumber
	AddBoardOnNNTPPost bool
}

// readonly for now

func NewPSQLIB(cfg Config) (p *PSQLIB, err error) {
	p = new(PSQLIB)

	st_once.Do(loadStatements)
	if st_loaderr != nil {
		return nil, st_loaderr
	}

	p.log = NewLogToX(*cfg.Logger, fmt.Sprintf("psqlib.%p", p))

	p.db = *cfg.DB

	p.src, err = fstore.OpenFStore(*cfg.SrcCfg)
	if err != nil {
		return nil, err
	}
	//p.src.CleanTemp()

	p.thm, err = fstore.OpenFStore(*cfg.ThmCfg)
	if err != nil {
		return nil, err
	}
	//p.thm.CleanTemp()

	p.nntpfs, err = fstore.OpenFStore(*cfg.NNTPFSCfg)
	if err != nil {
		return nil, err
	}
	//p.nntpfs.RemoveDir(nntpIncomingTempDir)
	p.nntpfs.MakeDir(nntpIncomingDir)
	p.nntpfs.MakeDir(nntpPullerDir)

	if cfg.TBuilder != nil {

		p.thumbnailer, err = cfg.TBuilder.BuildThumbnailer(&p.thm)
		if err != nil {
			return nil, err
		}

		p.tplan_thread = thumbnailer.ThumbPlan{
			Name:        "t",
			ThumbConfig: *cfg.TCfgThread,
		}
		p.tplan_reply = thumbnailer.ThumbPlan{
			Name:        "r",
			ThumbConfig: *cfg.TCfgReply,
		}
		if cfg.TCfgSage != nil {
			p.tplan_sage = thumbnailer.ThumbPlan{
				Name:        "s",
				ThumbConfig: *cfg.TCfgSage,
			}
		} else {
			p.tplan_sage = p.tplan_reply
		}

	} else {
		p.thumbnailer = nilthumbnailer.NilThumbnailer{}
	}

	p.nntpmgr = newNNTPCacheMgr()

	p.altthumb = *cfg.AltThumber

	p.ffo = formFileOpener{&p.src}

	p.fpp = form.DefaultParserParams
	// TODO make configurable
	p.fpp.MaxFileCount = 100
	p.fpp.MaxFileAllSize = 64 * 1024 * 1024
	p.instance = "nekochan"          // TODO config
	p.maxArticleBodySize = 256 << 20 // TODO config

	p.autoAddNNTPPostGroup = cfg.AddBoardOnNNTPPost

	p.ntStmts = make(map[int]*sql.Stmt)
	p.npStmts = make(map[npTuple]*sql.Stmt)

	return
}

func (sp *PSQLIB) Prepare() (err error) {
	err = sp.prepareStatements()
	if err != nil {
		return
	}

	return
}

func (dbib *PSQLIB) InitAndPrepare() (err error) {
	valid, err := dbib.CheckIb0()
	if err != nil {
		err = fmt.Errorf("error checking: %v", err)
		return
	}
	if !valid {
		dbib.log.LogPrint(NOTICE,
			"uninitialized PSQLIB db, attempting to initialize")

		dbib.InitIb0()

		valid, err = dbib.CheckIb0()
		if err != nil {
			err = fmt.Errorf("error checking (2): %v", err)
			return
		}
		if !valid {
			err = errors.New("database still not valid after initialization")
			return
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
