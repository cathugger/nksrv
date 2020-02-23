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

type regModInfo struct {
	modid uint64
	actionable bool

	ModCombinedCaps
}

func (ctx *wp_context) wp_registered_mod(tx *sql.Tx) (regModInfo, error) {
	ct := ctx.traceStart("wp_registered_mod %s", ctx.pubkeystr)
	defer ct.Done()

	return sp.registeredMod(tx, ctx.pubkeystr)
}

func (ctx *wp_context) wp_insertsql(tx *sql.Tx) (err error) {
	yct := ctx.traceStart("wp_insertsql %p", tx)
	defer yct.Done()

	err = sp.makeDelTables(tx)
	if err != nil {
		return
	}

	var rmi regModInfo
	if ctx.isctlgrp && ctx.pubkeystr != "" {
		rmi, err = ctx.wp_registered_mod(tx)
		if err != nil {
			return
		}
	}

	var gpid, bpid postID
	var duplicate bool
	// perform insert
	if !ctx.isReply {

		ct := ctx.traceStart("insert newthread post data to database")

		gpid, bpid, duplicate, err =
			ctx.sp.insertNewThread(tx, gstmt, bid, pInfo, isctlgrp, modid)

		ct.Done()

	} else {

		ct := ctx.traceStart("insert reply post data to database")

		gpid, bpid, duplicate, err =
			ctx.sp.insertNewReply(
				tx, gstmt,
				replyTargetInfo{bid, postID(tid.Int64)},
				pInfo, modid)

		ct.Done()

	}
	if err != nil {
		return
	}
	if duplicate {
		// shouldn't really happen...
		err = errDuplicateArticle
		return
	}

	// we've inserted file infos, so do P->A
	ctx.wp_act_fpp_bc_spawn_PA()

	// execute mod cmd
	if rmi.actionable {
		// we should execute it
		// we never put message in file when processing message

		ct := ctx.traceStart("execute mod cmd %s", pInfo.MessageID)

		_, err, _ =
			sp.execModCmd(
				tx, gpid, bid, bpid,
				modid, modCC,
				pInfo, nil, pInfo.MessageID,
				cref, delModIDState{})

		ct.Done()

		if err != nil {
			return
		}

	}

	err = sp.processRefsAfterPost(
		tx,
		ctx.srefs, irefs, inreplyto,
		bid, uint64(tid.Int64), bpid,
		pInfo.ID, board, pInfo.MessageID)

	if err != nil {
		return
	}
}

var sqlSerializedOpts = sql.TxOptions{Isolation: sql.LevelSerializable}

// isRetriableError returns true if error is sort-of expected and operation shall be retried.
func isRetriableError(err error) bool {
	var rerr psqlRetriableError
	return xerrors.As(err, &rerr)
}

func (ctx *wp_context) wp_act_commit() (err error) {

	yct := ctx.traceStart("wp_act_commit")
	defer yct.Done()

	// before-commit file postprocessing
	ctx.wp_act_fpp_bc_spawn_TP()
	defer func() {
		// if it haven't err'd then these must b already done
		if err != nil {
			// hold on incase we seriously fail before commit
			ctx.wg_TP.Wait()
			ctx.wg_PA.Wait()
		}
	}()

	numsoftfail := 0
	for {
		// do it inside inline func to allow defer
		func(){
			zct := ctx.traceStart("wp_act_commit whole tx")
			defer zct.Done()

			// start transaction
			var tx *sql.Tx
			tx, err = sp.db.DB.BeginTx(ctx.ctx, &sqlSerializedOpts)
			if err != nil {
				err = sp.sqlError("webpost tx begin", err)
				return
			}
			// if error, attempt rollback
			defer func() {
				if err != nil {
					ct := ctx.traceStart("wp_act_commit rollback")
					// error here isn't really relevant as long as we finish the operation
					_ = tx.Rollback()
					ct.Done()
				}
			}()

			// perform insertion operations
			err = ctx.wp_insertsql(tx)
			if err != nil {
				return
			}

			// before commit, ensure we've finished flushing files
			ct := ctx.traceStart("wp_act_commit wait files")
			ctx.wg_PA.Wait() // spawned inside wp_insertsql
			err = ctx.get_werr()
			ct.Done()
			if err != nil {
				// if file worker err'd, don't commit
				return
			}

			// commit
			ct = ctx.traceStart("wp_act_commit commit")
			err = tx.Commit()
			ct.Done()
			if err != nil {
				err = sp.sqlError("webpost tx commit", err)
				return
			}
		}()

		if err == nil || !isRetriableError(err) || numsoftfail >= 1000 {
			// if succeeded, or err'd in a way we don't allow, done here
			break
		}

		ctx.log.LogPrintf(DEBUG, "wp_act_commit retriable err: %v", err)

		// otherwise try again
		numsoftfail++
	}

	if err == nil {
		ctx.log.LogPrintf(DEBUG, "wp_act_commit finished loop without error")
	} else {
		ctx.log.LogPrintf(DEBUG, "wp_act_commit finished loop with error: %v", err)
	}
}
