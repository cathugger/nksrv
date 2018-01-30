package psqlib

// psql imageboard module

import (
	"../psqlcfg"
	"../psql"
	"../fstorecfg"
	"../fstore"
	"github.com/jmoiron/sqlx"
)

type PSQLIB struct {
	db psql.PSQL
	src fstore.FStore
	thumb fstore.FStore
}

type InitCfg struct {
	PSQLCfg psqlcfg.ConfigPSQL
	SrcCfg fstorecfg.ConfigFStore
	ThumbCfg fstorecfg.ConfigFStore
}

func NewPSQLIB(cfg InitCfg) (p PSQLIB, err error) {
	p.db, err = psql.OpenPSQL(cfg.PSQLCfg)
	if err != nil {
		return
	}
	p.src, err = fstore.OpenFStore(cfg.SrcCfg)
	if err != nil {
		return
	}
	p.thumb, err = fstore.OpenFStore(cfg.ThumbCfg)
	if err != nil {
		return
	}
	// TODO maybe some more initialization
	return
}
