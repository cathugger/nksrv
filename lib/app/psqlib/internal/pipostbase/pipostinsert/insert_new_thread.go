package pipostinsert

import (
	"database/sql"

	"github.com/lib/pq"

	"nksrv/lib/app/mailib"
	"nksrv/lib/utils/logx"

	"nksrv/lib/app/psqlib/internal/pibase"
	"nksrv/lib/app/psqlib/internal/pipostbase"
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
		sfs := makeSingleFileStuff(pInfo.FI[0])
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

				sfs.ftype,
				sfs.fsize,
				sfs.fid,
				sfs.fthumb,
				sfs.forig,
				sfs.fattrib,
				sfs.ftattrib,
				sfs.fextra)
	} else {
		mfs := makeMultiFileStuff(pInfo.FI)
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

				pq.Array(mfs.ftypes),
				pq.Array(mfs.fsizes),
				pq.Array(mfs.fids),
				pq.Array(mfs.fthumbs),
				pq.Array(mfs.forigs),
				pq.Array(mfs.fattribs),
				pq.Array(mfs.ftattribs),
				pq.Array(mfs.fextras))
	}

	sp.Log.LogPrintf(logx.DEBUG, "NEWTHREAD %s process", pInfo.ID)

	err = r.Scan(&gpid, &bpid)
	if err != nil {
		if sqlerrIsDuplicate(err) {
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

func InsertNewThreadMB(
	sp *pibase.PSQLIB, tx *sql.Tx,
	bids []pibase.TBoardID, pInfo mailib.PostInfo, skipover bool, modid uint64) (
	gpid pibase.TPostID, bpids []pibase.TPostID, duplicate bool, err error) {

	if len(pInfo.H) == 0 {
		panic("post should have header filled")
	}

	// XXX until I decide how else it should be
	var postids []string
	for range bids {
		postids = append(postids, pInfo.ID)
	}

	Hjson := pipostbase.MustMarshal(pInfo.H)
	GAjson := pipostbase.MustMarshal(pInfo.GA)
	Ljson := pipostbase.MustMarshal(&pInfo.L)
	Ejson := pipostbase.MustMarshal(&pInfo.E)
	BAjson := pipostbase.MustMarshal(pInfo.BA)

	smodid := sql.NullInt64{Int64: int64(modid), Valid: modid != 0}

	sp.Log.LogPrintf(logx.DEBUG, "NEWTHREAD %s start <%s>", pInfo.ID, pInfo.MessageID)

	var r *sql.Rows

	if len(pInfo.FI) == 0 {
		r, err = tx.
			Stmt(sp.StPrep[pibase.St_post_newthread_mb_nf]).
			Query(
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

				pq.Array(bids),
				pq.Array(postids),
				skipover,
				smodid,
				BAjson)
	} else if len(pInfo.FI) == 1 {
		sfs := makeSingleFileStuff(pInfo.FI[0])
		r, err = tx.
			Stmt(sp.StPrep[pibase.St_post_newthread_mb_sf]).
			Query(
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

				pq.Array(bids),
				pq.Array(postids),
				skipover,
				smodid,
				BAjson,

				sfs.ftype,
				sfs.fsize,
				sfs.fid,
				sfs.fthumb,
				sfs.forig,
				sfs.fattrib,
				sfs.ftattrib,
				sfs.fextra)
	} else {
		mfs := makeMultiFileStuff(pInfo.FI)
		r, err = tx.
			Stmt(sp.StPrep[pibase.St_post_newthread_mb_mf]).
			Query(
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

				pq.Array(bids),
				pq.Array(postids),
				skipover,
				smodid,
				BAjson,

				pq.Array(mfs.ftypes),
				pq.Array(mfs.fsizes),
				pq.Array(mfs.fids),
				pq.Array(mfs.fthumbs),
				pq.Array(mfs.forigs),
				pq.Array(mfs.fattribs),
				pq.Array(mfs.ftattribs),
				pq.Array(mfs.fextras))
	}

	if err != nil {
		if sqlerrIsDuplicate(err) {
			// duplicate
			return 0, nil, true, nil
		}
		err = sp.SQLError("newreplymb insert query", err)
		return
	}

	defer r.Close()

	sp.Log.LogPrintf(logx.DEBUG, "NEWTHREAD %s process", pInfo.ID)

	for r.Next() {
		var bpid pibase.TPostID
		err = r.Scan(&gpid, &bpid)
		if err != nil {
			if sqlerrIsDuplicate(err) {
				// duplicate
				return 0, nil, true, nil
			}
			err = sp.SQLError("newthreadmb insert scan", err)
			return
		}
		bpids = append(bpids, bpid)
	}
	if err = r.Err(); err != nil {
		if sqlerrIsDuplicate(err) {
			// duplicate
			return 0, nil, true, nil
		}
		err = sp.SQLError("newreplymb insert iteration", err)
		return
	}

	sp.Log.LogPrintf(logx.DEBUG, "NEWTHREAD %s done", pInfo.ID)

	// done
	return
}
