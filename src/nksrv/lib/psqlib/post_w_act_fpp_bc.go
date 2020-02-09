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


func (ctx *wp_context) wp_fpp_bc_movensync(
	errch chan<- error, pendir string,
	iterf func(func(fromfull, tofull string))) {

	// move & sync individual files
	var xg sync.WaitGroup
	iterf(func(fromfull, tofull string){
		xg.Add(1)
		go func(){
			defer xg.Done()

			// do sync of contents before move, as move should change only metadata,
			// and file with broken contents in pending folder could be harmful
			ctx.wp_syncfilename(fromfull)

			err2 := ctx.wp_movefile_fast(fromfull, tofull)
			if err2 != nil {
				sendError(errch, err2)
				return
			}
		}
	})
	xg.Wait()

	// once all files are moved & sync'd, sync dir they're in
	ctx.wp_syncdir(pendir)

	// sync parent of pending dir to ensure that it won't be gone
	ctx.wp_syncdir(path.Dir(pendir))
}

func (ctx *wp_context) wp_fpp_bc_files(errch chan<- error) {

	var err1 error
	ctx.src_pending, err1 = ctx.sp.src.NewDir("pending", "pw-", "")
	if err1 != nil {
		sendError(errch, err1)
		return
	}

	iterf := func(f func(string, string)) {
		x := 0
		for _, fieldname := range FileFields {
			files := ctx.f.Files[fieldname]
			for i := range files {
				fromfull := files[i].F.Name()
				tofull := filepath.Join(ctx.src_pending, ctx.pInfo.FI[x].ID)

				f(fromfull, tofull)

				x++
			}
		}
		if ctx.msgid != "" {
			tofull := filepath.Join(ctx.src_pending, ctx.pInfo.FI[x].ID)
			f(ctx.msgid, tofull)
			x++
		}
		if x != len(ctx.pInfo.FI) {
			panic("wtf")
		}
	}

	ctx.wp_fpp_bc_movensync(errch, ctx.src_pending, iterf)
}

func (ctx *wp_context) wp_fpp_bc_thumbs(errch chan<- error) {

	var err1 error
	ctx.thm_pending, err1 = ctx.sp.thm.NewDir("pending", "pw-", "")
	if err1 != nil {
		sendError(errch, err1)
		return
	}

	iterf := func(f func(string, string)) {
		for x := range ctx.thumbMoves {
			fromfull := thumbMoves[x].fulltmpname
			tofull := filepath.Join(ctx.thm_pending, thumbMoves[x].destname)
			f(fromfull, tofull)
		}
	}

	ctx.wp_fpp_bc_movensync(errch, ctx.thm_pending, iterf)
}

func (ctx *wp_context) wp_act_fpp_bc_afiw_any(
	errch chan<- error,
	fromdir, rootdir string, mover *fstore.Mover,
	func iterf(func(id string))) {

	//wg.Add(getlen())
	iterf(func(id string){
		fromfull := filepath.Join(fromdir, id)
		tofull := rootdir + ctx.pInfo.FI[x].ID

		/*
		go func(){
			defer wg.Done()
		*/

		e := mover.HardlinkOrCopyIfNeededStable(fromfull, tofull)
		if e != nil {
			sendError(errch, e)
		}
	})
}

func (ctx *wp_context) wp_act_fpp_bc_afiw_files(errch chan<- error) {
	fromdir := ctx.src_pending
	rootdir := ctx.sp.src.Main()
	mover := &ctx.sp.pending2src
	iterf := func(func(string) f) {
		for x := range ctx.pInfo.FI { f(ctx.pInfo.FI[x].ID) }
	}
	wp_act_fpp_bc_afiw_any(errch, fromdir, rootdir, mover, iterf)
}

func (ctx *wp_context) wp_act_fpp_bc_afiw_thumbs(errch chan<- error) {
	fromdir := ctx.thm_pending
	rootdir := ctx.sp.thm.Main()
	mover := &ctx.sp.pending2thm
	iterf := func(func(string) f) {
		for x := range ctx.thumbMoves { f(ctx.thumbMoves[x].destname) }
	}
	wp_act_fpp_bc_afiw_any(errch, fromdir, rootdir, mover, iterf)
}


func (ctx *wp_context) wp_act_fpp_bc_work(errch chan<- error) {

	ct := ctx.traceStart("wp_act_fpp_bc_work")
	defer ct.Done()

	var wg sync.WaitGroup

	wg.Add(2)

	go func(){
		defer wg.Done()
		ctx.wp_fpp_bc_files(errch)
	}()

	go func(){
		defer wg.Done()
		ctx.wp_fpp_bc_thumbs(errch)
	}()

	wg.Wait()

	// wait for DB fileinfo write
	// once that's done, nothing shuold be able to delete these files off disk
	ctx.fi_inserted_mu.Lock()
	for !ctx.fi_inserted {
		ctx.fi_inserted_cond.Wait()
	}
	ctx.fi_inserted_mu.Unlock()

	// once fileinfo is written out, push it to roots
	ctx.wp_act_fpp_bc_afiw_files(&wg, errch)
	ctx.wp_act_fpp_bc_afiw_thumbs(&wg, errch)
	wg.Wait()
}

// before commit, spawns work to be ran in parallel with sql insertion funcs
func (ctx *wp_context) wp_act_fpp_bc(
	wg *sync.WaitGroup, errch chan<- error) {

	ct := ctx.traceStart("wp_act_fpp_bc")
	defer ct.Done()

	wg.Add(1)
	go func (){
		defer wg.Done()
		ctx.wp_act_fpp_bc_work(errch)
	}
}
