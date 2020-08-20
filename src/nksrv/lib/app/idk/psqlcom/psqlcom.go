package psqlcom

import (
	"fmt"

	. "nksrv/lib/utils/logx"
	"nksrv/lib/app/base/psql"
)

type PSQLCOM struct {
	db  psql.PSQL
	log Logger
}

type InitCfg struct {
	Logger LoggerX
	DB     psql.PSQL
}

func NewPSQLIB(cfg InitCfg) (p *PSQLCOM, err error) {
	p = new(PSQLCOM)

	p.log = NewLogToX(cfg.Logger, fmt.Sprintf("psqlcom.%p", p))

	p.db = cfg.DB

	return
}
