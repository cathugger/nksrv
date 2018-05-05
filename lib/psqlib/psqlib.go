package psqlib

// psql imageboard module

import (
	"../altthumber"
	"../fstore"
	"../fstorecfg"
	. "../logx"
	"../psql"
	"fmt"
)

type PSQLIB struct {
	db       psql.PSQL
	log      Logger
	src      fstore.FStore
	thumb    fstore.FStore
	altthumb altthumber.AltThumber
}

type InitCfg struct {
	DB         psql.PSQL
	Logger     LoggerX
	SrcCfg     fstorecfg.ConfigFStore
	ThumbCfg   fstorecfg.ConfigFStore
	AltThumber altthumber.AltThumber
}

// readonly for now

func NewPSQLIB(cfg InitCfg) (p *PSQLIB, err error) {
	p = new(PSQLIB)

	p.log = NewLogToX(cfg.Logger, fmt.Sprintf("psqlib.%p", p))

	p.db = cfg.DB

	p.src, err = fstore.OpenFStore(cfg.SrcCfg)
	if err != nil {
		return nil, err
	}

	p.thumb, err = fstore.OpenFStore(cfg.ThumbCfg)
	if err != nil {
		return nil, err
	}

	p.altthumb = cfg.AltThumber

	return
}
