package psqlib

// psql imageboard module

import (
	"fmt"

	"nekochan/lib/altthumber"
	"nekochan/lib/fstore"
	. "nekochan/lib/logx"
	"nekochan/lib/psql"
)

type PSQLIB struct {
	db       psql.PSQL
	log      Logger
	src      fstore.FStore
	thm      fstore.FStore
	altthumb altthumber.AltThumber
}

type Config struct {
	DB         psql.PSQL
	Logger     LoggerX
	SrcCfg     fstore.Config
	ThmCfg     fstore.Config
	AltThumber altthumber.AltThumber
}

// readonly for now

func NewPSQLIB(cfg Config) (p *PSQLIB, err error) {
	p = new(PSQLIB)

	p.log = NewLogToX(cfg.Logger, fmt.Sprintf("psqlib.%p", p))

	p.db = cfg.DB

	p.src, err = fstore.OpenFStore(cfg.SrcCfg)
	if err != nil {
		return nil, err
	}

	p.thm, err = fstore.OpenFStore(cfg.ThmCfg)
	if err != nil {
		return nil, err
	}

	p.altthumb = cfg.AltThumber

	return
}
