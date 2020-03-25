package psqlib

import (
	"os"

	. "nksrv/lib/logx"
)

func (ctx *nntpPostCtx) pn_cleanup_on_err() {
	for _, fn := range ctx.tmpfns {
		os.Remove(fn)
	}
	for _, ti := range ctx.thminfos {
		os.Remove(ti.FullTmpName)
	}
}
