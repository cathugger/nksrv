package psqlib

import (
	"database/sql"

	. "nksrv/lib/logx"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
)

func (sp *PSQLIB) getPrePostInfo(
	isReply bool, board, thread string, postOpts PostOptions) (
	rInfo postedInfo,
	bid boardID, tid sql.NullInt64, ref sql.NullString,
	postLimits submissionLimits, opdate pq.NullTime,
	err error) {

	var jbPL xtypes.JSONText // board post limits
	var jbXL xtypes.JSONText // board newthread/reply limits
	var jtRL xtypes.JSONText // thread reply limits

	// get info about board, its limits and shit. does it even exists?
	if !isReply {

		// new thread

		err = sp.st_prep[st_web_prepost_newthread].
			QueryRow(board).
			Scan(&bid, &jbPL, &jbXL)
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

		rInfo.Board = board

		postLimits = defaultNewThreadSubmissionLimits

	} else {

		// new post

		err = sp.st_prep[st_web_prepost_newpost].
			QueryRow(board, thread).
			Scan(&bid, &jbPL, &jbXL, &tid, &jtRL, &ref, &opdate)
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
			bid, jbPL, jbXL, tid, jtRL, ref)

		rInfo.Board = board

		if tid.Int64 <= 0 {
			err = webNotFound(errNoSuchThread)
			return
		}

		rInfo.ThreadID = thread

		postLimits = defaultReplySubmissionLimits

	}

	err = sp.unmarshalBoardConfig(&postLimits, jbPL, jbXL)
	if err != nil {
		return
	}

	if postOpts.nolimit {
		// TODO check whether poster is privileged or something
		postLimits = maxSubmissionLimits
	}

	// apply instance-specific limit tweaks
	sp.applyInstanceSubmissionLimits(&postLimits, isReply, board)

	return
}
