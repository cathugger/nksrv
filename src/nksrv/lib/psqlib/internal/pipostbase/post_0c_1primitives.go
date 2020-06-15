package pipostbase

import (
	"fmt"
	"os"

	fu "nksrv/lib/fileutil"
	. "nksrv/lib/logx"

	"golang.org/x/xerrors"
)

type traceContext struct {
	ctx   *postCommonContext
	label string
	info  string
}

func (ctx *postCommonContext) traceStart(f string, args ...interface{}) *traceContext {
	c := &traceContext{ctx: ctx}
	c.label = fmt.Sprintf("TRACE %p", c)
	c.info = fmt.Sprintf(f, args...)

	c.ctx.sp.log.LogPrintf(DEBUG, "%s [START] %s", c.label, c.info)
	return c
}

func (c *traceContext) Done() {
	c.ctx.sp.log.LogPrintf(DEBUG, "%s [ END ] %s", c.label, c.info)
}

func (ctx *postCommonContext) wp_syncdir(sdir string) {
	if ctx.sp.noFileSync {
		return
	}

	ct := ctx.traceStart("wp_syncdir %q", sdir)
	defer ct.Done()

	err := fu.SyncDir(sdir)
	if err != nil {
		ctx.log.LogPrintf(WARN, "SyncDir %q fail: %v", sdir, err)
	}
}

func (ctx *postCommonContext) wp_syncfilename(fname string) {
	if ctx.sp.noFileSync {
		return
	}

	ct := ctx.traceStart("wp_syncfilename %q", fname)
	defer ct.Done()

	err := fu.SyncFileName(fname)
	if err != nil {
		ctx.log.LogPrintf(
			WARN, "SyncFileName %q fail: %v", fname, err)
	}
}

func (ctx *postCommonContext) wp_movefile_fast(from, to string) error {
	ct := ctx.traceStart("wp_movefile_fast %q -> %q", from, to)
	defer ct.Done()

	// TODO use something more optimized?
	err := os.Rename(from, to)
	if err != nil {
		return xerrors.Errorf("Rename (fast) %q -> %q fail: %v", from, to, err)
	}

	return nil
}

func (ctx *postCommonContext) wp_movefile_or_delet(from, to string) error {
	ct := ctx.traceStart("wp_movefile_noclobber %q -> %q", from, to)
	defer ct.Done()

	err := fu.RenameNoClobber(from, to)

	if err != nil {

		if os.IsExist(err) {

			ctx.log.LogPrintf(
				DEBUG, "RenameNoClobber %q -> %q did not overwrite existing", from, to)

			err = os.Remove(from)
			if err != nil {
				return xerrors.Errorf("Remove %q fail: %v", from, err)
			}

			return nil
		}

		return xerrors.Errorf("RenameNoClobber %q -> %q fail: %v", from, to, err)
	}

	return nil
}
