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


func (ctx *wp_context) wp_fpp_ac_files() error {

	srcdir := ctx.sp.src.Main()

	for x := range pInfo.FI {
		from := filepath.Join(ctx.src_pending, pInfo.FI[x].ID)
		to := srcdir + pInfo.FI[x].ID

		err := ctx.wp_movefile_or_delet(from, to)
		if err != nil {
			return err
		}
	}
}

func (ctx *wp_context) wp_fpp_ac_thumbs() error {

	thmdir := ctx.sp.thm.Main()

	for x := range thumbMoves {
		from := filepath.Join(ctx.thm_pending, thumbMoves[x].destname)
		to := thmdir + thumbMoves[x].destname

		err := ctx.wp_movefile_or_delet(from, to)
		if err != nil {
			return err
		}
	}
}

func (ctx *wp_context) wp_fpp_ac() (err error) {
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
