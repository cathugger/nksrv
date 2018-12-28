package psqlib

// psql imageboard module

import (
	"database/sql"
	"fmt"
	"sync"

	"centpd/lib/altthumber"
	"centpd/lib/fstore"
	. "centpd/lib/logx"
	"centpd/lib/mail/form"
	"centpd/lib/psql"
)

type PSQLIB struct {
	db                   psql.PSQL
	log                  Logger
	src                  fstore.FStore
	thm                  fstore.FStore
	nntpfs               fstore.FStore
	nntpmgr              nntpCacheMgr
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

	scraper_nonce int64
}

type Config struct {
	DB                 *psql.PSQL
	Logger             *LoggerX
	SrcCfg             *fstore.Config
	ThmCfg             *fstore.Config
	NNTPFSCfg          *fstore.Config
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

	err = p.prepareStatements()
	if err != nil {
		return
	}

	p.src, err = fstore.OpenFStore(*cfg.SrcCfg)
	if err != nil {
		return nil, err
	}
	p.src.CleanTemp()

	p.thm, err = fstore.OpenFStore(*cfg.ThmCfg)
	if err != nil {
		return nil, err
	}
	p.thm.CleanTemp()

	p.nntpfs, err = fstore.OpenFStore(*cfg.NNTPFSCfg)
	if err != nil {
		return nil, err
	}
	p.nntpfs.RemoveDir(nntpIncomingTempDir)
	p.nntpfs.MakeDir(nntpIncomingDir)
	p.nntpfs.MakeDir(nntpScraperDir)

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
