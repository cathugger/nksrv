package psqlib

import (
	"os"
	"path/filepath"
)

func (ctx *postWebContext) wp_fpp_ac_files() (err error) {

	// XXX we could replace this with RemoveAll I guess...

	if ctx.SrcPending == "" {
		// maybe it had no files, skip rest then
		return
	}

	for x := range ctx.pInfo.FI {

		from := filepath.Join(ctx.SrcPending, ctx.pInfo.FI[x].ID)

		err = os.Remove(from)
		if err != nil {
			return
		}
	}

	err = os.Remove(ctx.SrcPending)
	if err != nil {
		return
	}

	return
}

func (ctx *postWebContext) wp_fpp_ac_thumbs() (err error) {

	if ctx.ThmPending == "" {
		// maybe it had no thumbs, skip rest then
		return nil
	}

	for x := range ctx.ThumbInfos {
		from := filepath.Join(
			ctx.ThmPending, ctx.ThumbInfos[x].RelDestName)

		err = os.Remove(from)
		if err != nil {
			return
		}
	}

	err = os.Remove(ctx.ThmPending)
	if err != nil {
		return
	}

	return
}

// after commit
func (ctx *postWebContext) wp_act_fpp_ac() (err error) {

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
