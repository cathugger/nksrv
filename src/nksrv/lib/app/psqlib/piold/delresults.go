package psqlib

import "database/sql"

func (sp *PSQLIB) makeDelTables(tx *sql.Tx) (err error) {

	_, err = tx.Exec("CREATE TEMPORARY TABLE t_del_gposts (msgid TEXT NOT NULL) ON COMMIT DROP")
	if err != nil {
		err = sp.SQLError("", err)
		return
	}

	_, err = tx.Exec("CREATE TEMPORARY TABLE t_del_bposts (b_id INTEGER NOT NULL, p_name TEXT NOT NULL, msgid TEXT NOT NULL) ON COMMIT DROP")
	if err != nil {
		err = sp.SQLError("", err)
		return
	}

	_, err = tx.Exec("CREATE TEMPORARY TABLE t_del_modids (mod_id BIGINT NOT NULL) ON COMMIT DROP")
	if err != nil {
		err = sp.SQLError("", err)
		return
	}

	return
}

func (sp *PSQLIB) drainDelGPosts(tx *sql.Tx) (rows *sql.Rows, err error) {
	// we should prolly not touch this directly,
	// but notify using job queue + channel,
	// that will be guaranteed to invalidate in multidaemon cases,
	// and we probably could bolt some mechanism to async wait till it fully invalidates
	rows, err = tx.Query("DELETE FROM t_del_gposts RETURNING msgid")
	if err != nil {
		err = sp.SQLError("", err)
	}
	return
}

func (sp *PSQLIB) drainDelBPosts(tx *sql.Tx) (rows *sql.Rows, err error) {
	// XXX in idea we should limit somehow, but it's probably not important enough yet
	q := `WITH
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
	d.b_id = b.b_id`
	rows, err = tx.Query(q)
	if err != nil {
		err = sp.SQLError("", err)
	}
	return
}

func (sp *PSQLIB) drainDelModIDs(tx *sql.Tx) (rows *sql.Rows, err error) {
	rows, err = tx.Query("DELETE FROM t_del_modids RETURNING mod_id")
	if err != nil {
		err = sp.SQLError("", err)
	}
	return
}
