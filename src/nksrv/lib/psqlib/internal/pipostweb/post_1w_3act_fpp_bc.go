package pipostweb

import (
	"path/filepath"
	"sync"
)

func (ctx *postWebContext) wp_fpp_bc_hasfiles() bool {
	return len(ctx.pInfo.FI) != 0
}

func (ctx *postWebContext) wp_fpp_bc_files() {

	// make pending dir
	var err1 error
	ctx.src_pending, err1 = ctx.sp.src.NewDir(pendingDir, "pw-", "")
	if err1 != nil {
		ctx.set_werr(err1)
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
		if ctx.msgfn != "" {
			tofull := filepath.Join(ctx.src_pending, ctx.pInfo.FI[x].ID)
			f(ctx.msgfn, tofull)
			x++
		}
		if x != len(ctx.pInfo.FI) {
			panic("wtf")
		}
	}

	ctx.wp_fpp_bc_movensync(ctx.src_pending, iterf)
}

func (ctx *postWebContext) wp_fpp_bc_hasthumbs() bool {
	return len(ctx.thumbInfos) != 0
}

func (ctx *postWebContext) wp_fpp_bc_thumbs() {

	// make pending dir
	var err1 error
	ctx.thm_pending, err1 = ctx.sp.thm.NewDir(pendingDir, "pw-", "")
	if err1 != nil {
		ctx.set_werr(err1)
		return
	}

	iterf := func(f func(string, string)) {
		for x := range ctx.thumbInfos {
			fromfull := ctx.thumbInfos[x].FullTmpName
			tofull := filepath.Join(ctx.thm_pending, ctx.thumbInfos[x].RelDestName)
			f(fromfull, tofull)
		}
	}

	ctx.wp_fpp_bc_movensync(ctx.thm_pending, iterf)
}

func (ctx *postWebContext) wp_act_fpp_bc_afiw_files() {
	fromdir := ctx.src_pending
	rootdir := ctx.sp.src.Main()
	mover := &ctx.sp.pending2src
	iterf := func(f func(string)) {
		for x := range ctx.pInfo.FI {
			f(ctx.pInfo.FI[x].ID)
		}
	}
	ctx.wp_act_fpp_bc_afiw_any(fromdir, rootdir, mover, iterf)
}

func (ctx *postWebContext) wp_act_fpp_bc_afiw_thumbs() {
	fromdir := ctx.thm_pending
	rootdir := ctx.sp.thm.Main()
	mover := &ctx.sp.pending2thm
	iterf := func(f func(string)) {
		for x := range ctx.thumbInfos {
			f(ctx.thumbInfos[x].RelDestName)
		}
	}
	ctx.wp_act_fpp_bc_afiw_any(fromdir, rootdir, mover, iterf)
}

func (ctx *postWebContext) wp_act_fpp_bc_work_TP() {

	ct := ctx.traceStart("wp_act_fpp_bc_work_TP")
	defer ct.Done()

	var zg sync.WaitGroup

	zg.Add(1)
	go func() {
		defer zg.Done()
		ctx.wp_fpp_bc_files()
	}()

	if ctx.wp_fpp_bc_hasthumbs() {
		zg.Add(1)
		go func() {
			defer zg.Done()
			ctx.wp_fpp_bc_thumbs()
		}()
	}

	zg.Wait()
}

func (ctx *postWebContext) wp_act_fpp_bc_work_PA() {

	ct := ctx.traceStart("wp_act_fpp_bc_work_PA")
	defer ct.Done()

	// we're spawn once fileinfo is written to DB
	// once that's done, nothing shuold be able to delete these files off disk
	// we want T->P process to finish before we do our stuff
	ctx.wg_TP.Wait()

	if ctx.get_werr() != nil {
		// don't do anything if T->P err'd
		return
	}

	// stuff landed to P, push it to A
	ctx.wp_act_fpp_bc_afiw_files()
	ctx.wp_act_fpp_bc_afiw_thumbs()
}

// before commit, spawns work to be ran in parallel with sql insertion funcs
func (ctx *postWebContext) wp_act_fpp_bc_spawn_TP() {

	ct := ctx.traceStart("wp_act_fpp_bc_spawn_TP")
	defer ct.Done()

	if !ctx.wp_fpp_bc_hasfiles() {
		// if has no files, then has no thumbs too so moot
		return
	}

	ctx.wg_TP.Add(1)
	go func() {
		defer ctx.wg_TP.Done()
		ctx.wp_act_fpp_bc_work_TP()
	}()
}

func (ctx *postWebContext) wp_act_fpp_bc_spawn_PA() {

	ct := ctx.traceStart("wp_act_fpp_bc_spawn_PA")
	defer ct.Done()

	if !ctx.wp_fpp_bc_hasfiles() {
		// waste to spawn goroutine if we have nothing to do
		return
	}

	// ensure we don't have more than one of these
	ctx.wg_PA.Wait()

	ctx.wg_PA.Add(1)
	go func() {
		defer ctx.wg_PA.Done()
		ctx.wp_act_fpp_bc_work_PA()
	}()
}
