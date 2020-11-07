package pimod

import (
	"database/sql"

	. "nksrv/lib/app/psqlib/internal/pibase"
)

// makeDelTables creates tables for deletion triggers to store data in.
func (mc *modCtx) makeDelTables() (err error) {

	// deleted board posts
	// for references invalidation
	qDelBPosts := `
CREATE TEMPORARY TABLE
	t_del_bposts (
		b_id    INTEGER  NOT NULL,
		p_name  TEXT     NOT NULL,
		msgid   TEXT     NOT NULL
	)
ON COMMIT
	DROP
`
	_, err = mc.tx.Exec(qDelBPosts)
	if err != nil {
		err = mc.sp.SQLError("", err)
		return
	}

	// moderators ids whose posts were deleted.
	// used to trigger early discard of cached modposts as these modposts could be deleted.
	// XXX such invalidation could be implemented in more efficient way than this
	_, err = mc.tx.Exec("CREATE TEMPORARY TABLE t_del_modids (mod_id BIGINT NOT NULL) ON COMMIT DROP")
	if err != nil {
		err = mc.sp.SQLError("", err)
		return
	}

	return
}

// drainDelBPosts drains deleted bposts tmp table.
func (mc *modCtx) drainDelBPosts() (rows *sql.Rows, err error) {
	// XXX limit
	// do another query to set aside for later processing
	q := `
WITH
	d AS (
		DELETE FROM
			t_del_bposts
		RETURNING
			b_id,
			p_name,
			msgid
	)
SELECT
	b.b_name,
	d.p_name,
	d.msgid
FROM
	d
JOIN
	ib0.boards b
ON
	d.b_id = b.b_id
`
	rows, err = tx.Query(q)
	if err != nil {
		err = sp.SQLError("", err)
	}
	return
}

// DrainDelModIDs drains deleted mod ids tmp table.
func DrainDelModIDs(sp *PSQLIB, tx *sql.Tx) (rows *sql.Rows, err error) {
	rows, err = tx.Query("DELETE FROM t_del_modids RETURNING mod_id")
	if err != nil {
		err = sp.SQLError("", err)
	}
	return
}
