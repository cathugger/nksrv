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



func (ctx *wp_context) wp_fpp_bc_files(errch chan error) {

		var err1 error
		ctx.src_pending, err1 = ctx.sp.src.NewDir("pending", "wp-", "")
		if err1 != nil {
			sendError(errch, err1)
			return
		}

		// move & sync individual files
		{
			var xg sync.WaitGroup
			x := 0

			for _, fieldname := range FileFields {
				files := ctx.f.Files[fieldname]
				for i := range files {
					// need to copy data to use in goroutine
					from := files[i].F.Name()
					to := filepath.Join(ctx.src_pending, pInfo.FI[x].ID)

					xg.Add(1)
					go func(){
						defer xg.Done()

						err2 := ctx.wp_movefile_fast(from, to)
						if err2 != nil {
							sendError(errch, err2)
							return
						}

						ctx.wp_syncfilename(to)
					}

					x++
				}
			}

			if ctx.msgfn != "" {
				to := filepath.Join(ctx.src_pending, ctx.pInfo.FI[x].ID)

				xg.Add(1)
				go func(){
					defer xg.Done()

					err2 := ctx.wp_movefile_fast(ctx.msgfn, to)
					if err2 != nil {
						sendError(errch, err2)
						return
					}

					ctx.wp_syncfilename(to)
				}

				x++
			}

			if x != len(pInfo.FI) {
				panic(fmt.Errorf(
					"file number mismatch: have %d should have %d",
					x, len(pInfo.FI)))
			}

			xg.Wait()
		}

		// once all files are moved & sync'd, sync dir they're in
		ctx.wp_syncdir(ctx.src_pending)

		// sync parent of pending dir to ensure that it won't be gone
		ctx.wp_syncdir(path.Dir(ctx.src_pending))
}

func (ctx *wp_context) wp_fpp_bc_thumbs(errch chan error) {

	var err1 error
	ctx.thm_pending, err1 = ctx.sp.thm.NewDir("pending", "wp-", "")
	if err1 != nil {
		sendError(errch, err1)
		return
	}

	// move & sync individual thumbs
	{
		var xg sync.WaitGroup
		for x := range ctx.thumbMoves {
			// need to copy data to use in goroutine
			from := thumbMoves[x].fulltmpname
			to := filepath.Join(ctx.thm_pending, thumbMoves[x].destname)

			xg.Add(1)
			go func(){
				defer xg.Done()

				err2 := ctx.wp_movefile_fast(from, to)
				if err2 != nil {
					sendError(errch, err2)
					return
				}

				ctx.wp_syncfilename(to)
			}
		}
		xg.Wait()
	}

	// once all files are moved & sync'd, sync dir they're in
	ctx.wp_syncdir(ctx.thm_pending)

	// sync parent of pending dir to ensure that it won't be gone
	ctx.wp_syncdir(path.Dir(ctx.thm_pending))
}

func (ctx *wp_context) wp_fpp_bc(wg *sync.WaitGroup, errch chan error) {

	wg.Add(2)

	go func(){
		defer wg.Done()
		ctx.wp_fpp_bc_files(errch)
	}()

	go func(){
		defer wg.Done()
		ctx.wp_fpp_bc_thumbs(errch)
	}()
}
