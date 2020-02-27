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


func (ctx *wp_context) wp_fpp_ac_files() (err error) {

	// XXX we could replace this with RemoveAll I guess...

	if ctx.src_pending == "" {
		// maybe it had no files, skip rest then
		return
	}

	for x := range pInfo.FI {

		from := filepath.Join(ctx.src_pending, ctx.pInfo.FI[x].ID)

		err = os.Remove(from)
		if err != nil {
			return
		}
	}

	err = os.Remove(ctx.src_pending)
	if err != nil {
		return
	}

	return
}

func (ctx *wp_context) wp_fpp_ac_thumbs() (err error) {

	// XXX we could replace this with RemoveAll I guess...

	if ctx.thm_pending == "" {
		// maybe it had no thumbs, skip rest then
		return nil
	}

	for x := range ctx.thumbMoves {

		from := filepath.Join(
			ctx.thm_pending, ctx.thumbMoves[x].destname)

		err = os.Remove(from)
		if err != nil {
			return
		}
	}

	err = os.Remove(ctx.thm_pending)
	if err != nil {
		return
	}

	return
}

// after commit
func (ctx *wp_context) wp_act_fpp_ac() (err error) {

	yct := ctx.traceStart("wp_act_fpp_ac")
	defer yct.Done()

	err = ctx.wp_fpp_ac_files()
	if err != nil {
		return
	}
	err = ctx.wp_fpp_ac_thumbs()
	if err != nil {
		return
	}
	return
}
