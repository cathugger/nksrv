package pipostbase

import (
	xtypes "github.com/jmoiron/sqlx/types"

	"nksrv/lib/psqlib/internal/pibase"
	"nksrv/lib/psqlib/internal/pibaseweb"
)

type (
	boardID = pibase.TBoardID
	postID  = pibase.TPostID
)

func unmarshalBoardConfig(
	sp *pibase.PSQLIB,
	postLimits *pibaseweb.SubmissionLimits, jbPL, jbXL xtypes.JSONText) (err error) {

	// jbPL - b.post_limits in all cases
	err = jbPL.Unmarshal(postLimits)
	if err != nil {
		return sp.SQLError("jbPL json unmarshal", err)
	}

	// jbXL - either b.newthread_limits or b.reply_limits
	err = jbXL.Unmarshal(postLimits)
	if err != nil {
		return sp.SQLError("jbXL json unmarshal", err)
	}

	return
}

func unmarshalBoardThreadOpts(
	sp *pibase.PSQLIB,
	threadOpts *pibaseweb.ThreadOptions, jbTO, jtTO xtypes.JSONText) (err error) {

	// jbTO - b.thread_opts
	err = jbTO.Unmarshal(threadOpts)
	if err != nil {
		return sp.SQLError("jbTO json unmarshal", err)
	}

	// jtTO - t.thread_opts
	err = jtTO.Unmarshal(threadOpts)
	if err != nil {
		return sp.SQLError("jtTO json unmarshal", err)
	}

	return
}

func unmarshalThreadConfig(
	sp *pibase.PSQLIB,
	postLimits *pibaseweb.SubmissionLimits, threadOpts *pibaseweb.ThreadOptions,
	jtRL, jbTO, jtTO xtypes.JSONText) (err error) {

	// jtRL - t.reply_limits
	err = jtRL.Unmarshal(postLimits)
	if err != nil {
		return sp.SQLError("jtRL json unmarshal", err)
	}

	return unmarshalBoardThreadOpts(sp, threadOpts, jbTO, jtTO)
}
