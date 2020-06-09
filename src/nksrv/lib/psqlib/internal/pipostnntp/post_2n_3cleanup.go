package psqlib

import (
	"os"
)

func (ctx *postNNTPContext) pn_cleanup_on_err() {
	for _, fn := range ctx.tmpfns {
		os.Remove(fn)
	}
	for _, ti := range ctx.thumbInfos {
		os.Remove(ti.FullTmpName)
	}
}
