package psqlib

import (
	"database/sql"

	. "nksrv/lib/logx"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
)

func (sp *PSQLIB) maybeTxStmt(tx *sql.Tx, stmt int) (r *sql.Stmt) {
	r = sp.st_prep[st_web_prepost_newthread]
	if tx != nil {
		r = tx.Stmt(r)
	}
	return
}

func (sp *PSQLIB) getPrePostInfo(
	tx *sql.Tx, btr wp_btr, postOpts PostOptions) (
	rInfo postedInfo, dbi wp_dbinfo, err error) {

	var jbPL xtypes.JSONText // board post limits
	var jbXL xtypes.JSONText // board newthread/reply limits
	var jtRL xtypes.JSONText // thread reply limits

	// get info about board, its limits and shit. does it even exists?
	if !btr.isReply {

		// new thread

		err = sp.maybeTxStmt(tx, st_web_prepost_newthread).
			QueryRow(btr.board).
			Scan(&dbi.bid, &jbPL, &jbXL)
		if err != nil {
			if err == sql.ErrNoRows {
				err = webNotFound(errNoSuchBoard)
				return
			}
			err = sp.sqlError("board row query scan", err)
			return
		}

		sp.log.LogPrintf(DEBUG,
			"got bid(%d) post_limits(%q) newthread_limits(%q)",
			bid, jbPL, jbXL)

		rInfo.Board = btr.board

		dbi.postLimits = defaultNewThreadSubmissionLimits

	} else {

		// new post

		err = sp.maybeTxStmt(tx, st_web_prepost_newpost).
			QueryRow(btr.board, btr.thread).
			Scan(&dbi.bid, &jbPL, &jbXL, &dbi.tid, &jtRL, &dbi.ref, &dbi.opdate)
		if err != nil {
			if err == sql.ErrNoRows {
				err = webNotFound(errNoSuchBoard)
				return
			}
			err = sp.sqlError("board x thread row query scan", err)
			return
		}

		sp.log.LogPrintf(DEBUG,
			"got bid(%d) b.post_limits(%q) b.reply_limits(%q) tid(%#v) "+
				"t.reply_limits(%q) p.msgid(%#v)",
			dbi.bid, jbPL, jbXL, dbi.tid, jtRL, dbi.ref)

		rInfo.Board = btr.board

		if dbi.tid.Int64 <= 0 {
			err = webNotFound(errNoSuchThread)
			return
		}

		rInfo.ThreadID = thread

		dbi.postLimits = defaultReplySubmissionLimits

	}

	err = sp.unmarshalBoardConfig(&dbi.postLimits, jbPL, jbXL)
	if err != nil {
		return
	}

	if postOpts.nolimit {
		// TODO check whether poster is privileged or something
		dbi.postLimits = maxSubmissionLimits
	}

	// apply instance-specific limit tweaks
	sp.applyInstanceSubmissionLimits(&dbi.postLimits, btr.isReply, btr.board)

	return
}
