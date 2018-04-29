package psqlib

// psql imageboard module

import (
	"../altthumber"
	"../fstore"
	"../fstorecfg"
	. "../logx"
	"../psql"
	"../psqlcfg"
	"fmt"
)

type PSQLIB struct {
	db       psql.PSQL
	src      fstore.FStore
	thumb    fstore.FStore
	log      Logger
	altthumb altthumber.AltThumber
}

type InitCfg struct {
	PSQLCfg    psqlcfg.ConfigPSQL
	SrcCfg     fstorecfg.ConfigFStore
	ThumbCfg   fstorecfg.ConfigFStore
	Logger     LoggerX
	AltThumber altthumber.AltThumber
}

// readonly for now

func NewPSQLIB(cfg InitCfg) (p *PSQLIB, err error) {
	p = new(PSQLIB)
	p.log = NewLogToX(cfg.Logger, fmt.Sprintf("psqlib.%p", p))
	p.db, err = psql.OpenPSQL(cfg.PSQLCfg)
	if err != nil {
		return nil, err
	}
	p.src, err = fstore.OpenFStore(cfg.SrcCfg)
	if err != nil {
		return nil, err
	}
	p.thumb, err = fstore.OpenFStore(cfg.ThumbCfg)
	if err != nil {
		return nil, err
	}
	p.altthumb = cfg.AltThumber
	// TODO maybe some more initialization
	return
}
