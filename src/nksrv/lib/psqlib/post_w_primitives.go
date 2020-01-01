package psqlib

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/ed25519"

	"nksrv/lib/date"
	fu "nksrv/lib/fileutil"
	"nksrv/lib/ibref_nntp"
	. "nksrv/lib/logx"
	"nksrv/lib/mail"
	"nksrv/lib/mail/form"
	"nksrv/lib/mailib"
	tu "nksrv/lib/textutils"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/webcaptcha"
	ib0 "nksrv/lib/webib0"
)


func (ctx *wp_context) wp_syncdir(dir string) {
	if ctx.sp.noFileSync {
		return
	}

	ct := ctx.traceStart("wp_syncdir %q", dir)
	defer ct.Done()

	err := fu.SyncDir(sdir)
	if err != nil {
		ctx.log.LogPrintf(WARN, "SyncDir %q fail: %v", dir, err)
	}
}

func (ctx *wp_context) wp_syncfilename(fname string) {
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

func (ctx *wp_context) wp_movefile_fast(from, to string) error {
	ct := ctx.traceStart("wp_movefile_fast %q -> %q", from, to)
	defer ct.Done()

	// TODO use something more optimized?
	err := os.Rename(from, to)
	if err != nil {
		return xerrors.Errorf("Rename (fast) %q -> %q fail: %v", from, to, err)
	}

	return nil
}

func (ctx *wp_context) wp_movefile_or_delet(from, to string) error {
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
