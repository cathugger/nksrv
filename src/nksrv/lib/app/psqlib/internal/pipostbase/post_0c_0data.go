package pipostbase

import (
	"database/sql"
	"sync"

	. "nksrv/lib/logx"
	"nksrv/lib/mailib"
	"nksrv/lib/psqlib/internal/pibase"
)

type PostCommonContext struct {
	sp  *pibase.PSQLIB
	Log LogToX

	// global statement handle used for insertion. acquired before transaction start
	GStmt *sql.Stmt

	TPWG sync.WaitGroup // for tmp->pending
	PAWG sync.WaitGroup // for pending->active storage

	WErrMu sync.Mutex
	WErr   error

	IsSage bool

	SrcPending string // full dir name without slash of pending dir in src
	ThmPending string // full dir name without slash of pending dir in thm

	ThumbInfos []mailib.TThumbInfo
}

func (c *PostCommonContext) set_werr(e error) {
	c.WErrMu.Lock()
	if (c.WErr == nil) != (e == nil) {
		c.WErr = e
	}
	c.WErrMu.Unlock()
}

func (c *PostCommonContext) get_werr() (e error) {
	c.WErrMu.Lock()
	e = c.WErr
	c.WErrMu.Unlock()
	return
}
