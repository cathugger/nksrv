package pipostbase

import (
	"database/sql"
	"sync"

	. "nksrv/lib/logx"
	"nksrv/lib/mailib"
)

type postCommonContext struct {
	sp  *PSQLIB
	log LogToX

	// global statement handle used for insertion. acquired before transaction start
	gstmt *sql.Stmt

	wg_TP sync.WaitGroup // for tmp->pending
	wg_PA sync.WaitGroup // for pending->active storage

	werr_mu sync.Mutex
	werr    error

	isSage bool

	src_pending string // full dir name without slash of pending dir in src
	thm_pending string // full dir name without slash of pending dir in thm

	thumbInfos []mailib.TThumbInfo
}

func (c *postCommonContext) set_werr(e error) {
	c.werr_mu.Lock()
	if (c.werr == nil) != (e == nil) {
		c.werr = e
	}
	c.werr_mu.Unlock()
}

func (c *postCommonContext) get_werr() (e error) {
	c.werr_mu.Lock()
	e = c.werr
	c.werr_mu.Unlock()
	return
}
