package psqlib

import (
	"database/sql"
	"errors"
)

func (sp *PSQLIB) modset_processOneJob() (hadwork bool, err error) {

	tx, err := sp.db.DB.Begin()
	if err != nil {
		err = sp.sqlError("begin tx", err)
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	mcg := tx.Stmt(sp.st_prep[st_mod_joblist_modlist_changes_get])

	var (
		j_id   uint64
		mod_id uint64

		t_date_sent sql.NullTime
		t_g_p_id    sql.NullInt64
		t_b_id      sql.NullInt32
	)

	err = mcg.QueryRow().Scan(&j_id, &mod_id, &t_date_sent, &t_g_p_id, &t_b_id)
	if err != nil {
		if err == sql.ErrNoRows {
			// overwrite err
			err = tx.Commit()
			if err != nil {
				err = sp.sqlError("commit tx", err)
			}
			return
		}

		err = sp.sqlError("queryrowscan", err)
		return
	}

	// TODO process some msgs then commit back
	err = errors.New("kill me")
	panic("I need dye")
}

// reads jobs n shit

func (sp *PSQLIB) modset_jobprocessor(notif <-chan struct{}) {
	doscan := true
	for {
		if !doscan {
			// wait
			_, ok := <-notif
			if !ok {
				return
			}
		}
		///
	}
}
