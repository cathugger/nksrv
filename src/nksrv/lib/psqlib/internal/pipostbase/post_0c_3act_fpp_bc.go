package pipostbase

import (
	"path"
	"path/filepath"
	"sync"

	"nksrv/lib/fstore"
)

func (ctx *PostCommonContext) wp_fpp_bc_movensync(
	pendir string, iterf func(func(fromfull, tofull string))) {

	// move & sync individual files
	var xg sync.WaitGroup
	iterf(func(fromfull, tofull string) {
		xg.Add(1)
		go func() {
			defer xg.Done()

			// do sync of contents before move, as move should change only metadata,
			// and file with broken contents in pending folder could be harmful
			ctx.wp_syncfilename(fromfull)

			err2 := ctx.wp_movefile_fast(fromfull, tofull)
			if err2 != nil {
				ctx.set_werr(err2)
				return
			}
		}()
	})
	xg.Wait()

	// once all files are moved & sync'd, sync dir they're in
	ctx.wp_syncdir(pendir)

	// sync parent of pending dir to ensure that it won't be gone
	ctx.wp_syncdir(path.Dir(pendir))
}

func (ctx *PostCommonContext) wp_act_fpp_bc_afiw_any(
	fromdir, rootdir string, mover *fstore.Mover,
	iterf func(func(id string))) {

	iterf(func(id string) {
		fromfull := filepath.Join(fromdir, id)
		tofull := rootdir + id

		e := mover.HardlinkOrCopyIfNeededStable(fromfull, tofull)
		if e != nil {
			ctx.set_werr(e)
		}
	})
}
