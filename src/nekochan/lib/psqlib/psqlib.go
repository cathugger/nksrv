package psqlib

// psql imageboard module

import (
	"fmt"

	"nekochan/lib/altthumber"
	"nekochan/lib/fstore"
	. "nekochan/lib/logx"
	"nekochan/lib/mail/form"
	"nekochan/lib/psql"
)

type PSQLIB struct {
	db       psql.PSQL
	log      Logger
	src      fstore.FStore
	thm      fstore.FStore
	altthumb altthumber.AltThumber
	ffo      formFileOpener
	fpp      form.ParserParams
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
	p.src.Clean()

	p.thm, err = fstore.OpenFStore(cfg.ThmCfg)
	if err != nil {
		return nil, err
	}
	p.thm.Clean()

	p.altthumb = cfg.AltThumber

	p.ffo = formFileOpener{&p.src}

	p.fpp = form.DefaultParserParams
	// TODO make configurable
	p.fpp.MaxFileCount = 100
	p.fpp.MaxFileAllSize = 64 * 1024 * 1024

	return
}
