package psqlib

import (
	xtypes "github.com/jmoiron/sqlx/types"
)

func (sp *PSQLIB) unmarshalBoardConfig(
	postLimits *submissionLimits, jbPL, jbXL xtypes.JSONText) (err error) {

	// jbPL - b.post_limits in all cases
	err = jbPL.Unmarshal(postLimits)
	if err != nil {
		return sp.sqlError("jbPL json unmarshal", err)
	}

	// jbXL - either b.newthread_limits or b.reply_limits
	err = jbXL.Unmarshal(postLimits)
	if err != nil {
		return sp.sqlError("jbXL json unmarshal", err)
	}

	return
}

func (sp *PSQLIB) unmarshalBoardThreadOpts(
	threadOpts *threadOptions, jbTO, jtTO xtypes.JSONText) (err error) {

	// jbTO - b.thread_opts
	err = jbTO.Unmarshal(threadOpts)
	if err != nil {
		return sp.sqlError("jbTO json unmarshal", err)
	}

	// jtTO - t.thread_opts
	err = jtTO.Unmarshal(threadOpts)
	if err != nil {
		return sp.sqlError("jtTO json unmarshal", err)
	}

	return
}

func (sp *PSQLIB) unmarshalThreadConfig(
	postLimits *submissionLimits, threadOpts *threadOptions,
	jtRL, jbTO, jtTO xtypes.JSONText) (err error) {

	// jtRL - t.reply_limits
	err = jtRL.Unmarshal(postLimits)
	if err != nil {
		return sp.sqlError("jtRL json unmarshal", err)
	}

	return sp.unmarshalBoardThreadOpts(threadOpts, jbTO, jtTO)
}
