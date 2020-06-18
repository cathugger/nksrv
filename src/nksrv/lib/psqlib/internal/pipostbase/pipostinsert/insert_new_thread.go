package pipostinsert

import (
	"database/sql"

	"github.com/lib/pq"

	"nksrv/lib/logx"
	"nksrv/lib/mailib"
	"nksrv/lib/psqlib/internal/pibase"
	"nksrv/lib/psqlib/internal/pipostbase"
)

func InsertNewThread(
	sp *pibase.PSQLIB, tx *sql.Tx,
	bid pibase.TBoardID, pInfo mailib.PostInfo, skipover bool, modid uint64) (
	gpid pibase.TPostID, bpid pibase.TPostID, duplicate bool, err error) {

	if len(pInfo.H) == 0 {
		panic("post should have header filled")
	}

	Hjson := pipostbase.MustMarshal(pInfo.H)
	GAjson := pipostbase.MustMarshal(pInfo.GA)
	Ljson := pipostbase.MustMarshal(&pInfo.L)
	Ejson := pipostbase.MustMarshal(&pInfo.E)
	BAjson := pipostbase.MustMarshal(pInfo.BA)

	smodid := sql.NullInt64{Int64: int64(modid), Valid: modid != 0}

	sp.Log.LogPrintf(logx.DEBUG, "NEWTHREAD %s start <%s>", pInfo.ID, pInfo.MessageID)

	var r *sql.Row

	if len(pInfo.FI) == 0 {
		r = tx.
			Stmt(sp.StPrep[pibase.St_post_newthread_sb_nf]).
			QueryRow(
				pInfo.Date,
				pInfo.MI.Sage,
				pInfo.FC,
				pInfo.MessageID,
				pInfo.MI.Title,
				pInfo.MI.Author,
				pInfo.MI.Trip,
				pInfo.MI.Message,
				Hjson,
				GAjson,
				Ljson,
				Ejson,

				bid,
				pInfo.ID,
				skipover,
				smodid,
				BAjson)
	} else if len(pInfo.FI) == 1 {
		r = tx.
			Stmt(sp.StPrep[pibase.St_post_newthread_sb_sf]).
			QueryRow(
				pInfo.Date,
				pInfo.MI.Sage,
				pInfo.FC,
				pInfo.MessageID,
				pInfo.MI.Title,
				pInfo.MI.Author,
				pInfo.MI.Trip,
				pInfo.MI.Message,
				Hjson,
				GAjson,
				Ljson,
				Ejson,

				bid,
				pInfo.ID,
				skipover,
				smodid,
				BAjson,

				pInfo.FI[0].Type.String(),
				pInfo.FI[0].Size,
				pInfo.FI[0].ID,
				pInfo.FI[0].ThumbField,
				pInfo.FI[0].Original,
				pipostbase.MustMarshal(pInfo.FI[0].FileAttrib),
				pipostbase.MustMarshal(pInfo.FI[0].ThumbAttrib),
				pipostbase.MustMarshal(pInfo.FI[0].Extras))
	} else {
		var ftypes []string
		var fsizes []int64
		var fids []string
		var fthumbs []string
		var forigs []string
		var fattribs [][]byte
		var ftattribs [][]byte
		var fextras [][]byte

		for i := range pInfo.FI {
			ftypes = append(ftypes,
				pInfo.FI[i].Type.String())
			fsizes = append(fsizes,
				pInfo.FI[i].Size)
			fids = append(fids,
				pInfo.FI[i].ID)
			fthumbs = append(fthumbs,
				pInfo.FI[i].ThumbField)
			forigs = append(forigs,
				pInfo.FI[i].Original)
			fattribs = append(fattribs,
				pipostbase.MustMarshal(pInfo.FI[i].FileAttrib))
			ftattribs = append(ftattribs,
				pipostbase.MustMarshal(pInfo.FI[i].ThumbAttrib))
			fextras = append(fextras,
				pipostbase.MustMarshal(pInfo.FI[i].Extras))
		}

		r = tx.
			Stmt(sp.StPrep[pibase.St_post_newthread_sb_mf]).
			QueryRow(
				pInfo.Date,
				pInfo.MI.Sage,
				pInfo.FC,
				pInfo.MessageID,
				pInfo.MI.Title,
				pInfo.MI.Author,
				pInfo.MI.Trip,
				pInfo.MI.Message,
				Hjson,
				GAjson,
				Ljson,
				Ejson,

				bid,
				pInfo.ID,
				skipover,
				smodid,
				BAjson,

				pq.Array(ftypes),
				pq.Array(fsizes),
				pq.Array(fids),
				pq.Array(fthumbs),
				pq.Array(forigs),
				pq.Array(fattribs),
				pq.Array(ftattribs),
				pq.Array(fextras))
	}

	sp.Log.LogPrintf(logx.DEBUG, "NEWTHREAD %s process", pInfo.ID)

	err = r.Scan(&gpid, &bpid)
	if err != nil {
		if pqerr, ok := err.(*pq.Error); ok && pqerr.Code == "23505" {
			// duplicate
			return 0, 0, true, nil
		}
		err = sp.SQLError("newthread insert query scan", err)
		return
	}

	sp.Log.LogPrintf(logx.DEBUG, "NEWTHREAD %s done", pInfo.ID)

	// done
	return
}
