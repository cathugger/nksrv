package psqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"nksrv/lib/captchastore"
	"nksrv/lib/date"
	. "nksrv/lib/logx"
	"nksrv/lib/psql"
)

const currDbVersion = "0"

type PSQLStore struct {
	db  *psql.PSQL
	log Logger
}

func (p PSQLStore) InitDb() (err error) {
	stmts := [...]string{
		`CREATE SCHEMA captcha`,
		`CREATE TABLE captcha.keks (
	kek_order    INTEGER GENERATED ALWAYS AS IDENTITY,
	kek_id       BIGINT  NOT NULL,
	kek_data     BYTEA   NOT NULL,
	kek_disabled BOOLEAN NOT NULL DEFAULT FALSE,

	PRIMARY KEY (kek_id),
	UNIQUE (kek_order)
)`,
		`CREATE TABLE captcha.solved (
	solved_key BYTEA                    NOT NULL,
	solved_exp TIMESTAMP WITH TIME ZONE NOT NULL,

	PRIMARY KEY (solved_key)
)`,
		`CREATE INDEX
	ON captcha.solved (solved_exp)`,
	}

	tx, err := p.db.DB.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if err != nil {
		return fmt.Errorf("err on BeginTx: %v", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for i, s := range stmts {
		_, err = tx.Exec(s)
		if err != nil {
			err = fmt.Errorf("err on stmt %d: %v", i, err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("err on Commit: %v", err)
	}
	return
}

func (p PSQLStore) CheckDb() (initialised bool, err error) {
	q := `SELECT version FROM capabilities WHERE component = 'captcha' LIMIT 1`
	var ver string
	err = p.db.DB.QueryRow(q).Scan(&ver)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, p.sqlError("version row query", err)
	}

	if ver != currDbVersion {
		return true, fmt.Errorf("incorrect ib0 schema version: %q (our: %q)", ver, currDbVersion)
	}

	return true, nil
}

func NewPSQLStore(db *psql.PSQL, l LoggerX) PSQLStore {
	log := NewLogToX(l, fmt.Sprintf("captchastore/psqlstore.%p", db))
	return PSQLStore{db: db, log: log}
}

func (p PSQLStore) InitAndPrepare() (err error) {
	valid, err := p.CheckDb()
	if err != nil {
		return fmt.Errorf("error checking: %v", err)
	}
	if !valid {
		p.log.LogPrint(NOTICE,
			"uninitialized PSQLIB db, attempting to initialize")

		err = p.InitDb()
		if err != nil {
			return fmt.Errorf("error initializing: %v", err)
		}

		valid, err = p.CheckDb()
		if err != nil {
			return fmt.Errorf("error checking (2): %v", err)
		}
		if !valid {
			return errors.New("database still not valid after initialization")
		}
	}

	return
}

func NewInitAndPrepare(db *psql.PSQL, l LoggerX) (p PSQLStore, err error) {
	p = NewPSQLStore(db, l)

	err = p.InitAndPrepare()
	if err != nil {
		return
	}

	return
}

func (p PSQLStore) StoreSolved(
	obj []byte, expires, nowtime int64) (fresh bool, err error) {

	exp_t := date.UnixTimeUTC(expires)
	now_t := date.UnixTimeUTC(nowtime)

	q := `WITH
	x AS (
		DELETE FROM
			captcha.solved
		WHERE
			solved_exp < $3
	)
INSERT INTO captcha.solved (solved_key,solved_exp) VALUES ($1,$2) ON CONFLICT (solved_key) DO NOTHING RETURNING TRUE`
	var inserted bool
	err = p.db.DB.QueryRow(q, obj, exp_t, now_t).Scan(&inserted)
	if err != nil {
		if err == sql.ErrNoRows {
			// conflict - already exists
			return false, nil
		}
		return false, p.sqlError("captcha solved queryrow scan", err)
	}
	if !inserted {
		// shouldn't happen
		return false, p.sqlError("bad inserted value", err)
	}
	return true, nil
}

type queryer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

func (p PSQLStore) loadKEKs(
	qq queryer, keks []captchastore.KEKInfo) ([]captchastore.KEKInfo, error) {

	q := `SELECT kek_id,kek_data,kek_disabled FROM captcha.keks ORDER BY kek_order DESC`
	rows, err := qq.Query(q)
	if err != nil {
		return keks, p.sqlError("kek query", err)
	}
	for rows.Next() {
		var kek captchastore.KEKInfo
		err = rows.Scan(&kek.ID, &kek.KEK, &kek.Disabled)
		if err != nil {
			return keks, p.sqlError("kek query rows scan", err)
		}
		keks = append(keks, kek)
	}
	if err = rows.Err(); err != nil {
		return keks, p.sqlError("kek query rows iteration", err)
	}
	return keks, nil
}

func hasValidKEKs(keks []captchastore.KEKInfo) bool {
	for i := range keks {
		if !keks[i].Disabled {
			return true
		}
	}
	return false
}

func (p PSQLStore) LoadKEKs(
	ifempty func() (id uint64, kek []byte)) (
	keks []captchastore.KEKInfo, err error) {

	keks, err = p.loadKEKs(p.db.DB, keks)
	if err != nil || ifempty == nil || hasValidKEKs(keks) {
		return
	}

	// currently no valid keks - do tx
	tx, err := p.db.DB.Begin()
	if err != nil {
		return nil, p.sqlError("kek tx begin", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// do explicit lock to prevent concurrent access
	_, err = tx.Exec("LOCK captcha.keks IN ACCESS EXCLUSIVE MODE")
	if err != nil {
		err = p.sqlError("lock exec", err)
		return
	}

	// recheck
	keks = keks[:0]
	keks, err = p.loadKEKs(tx, keks)
	if err != nil {
		return
	}
	if hasValidKEKs(keks) {
		_ = tx.Rollback() // we didn't modify anything so discard
		return
	}

	// so we have exclusive access and there are no valid keks at this point
	nfail := 0
	for {
		id, kek := ifempty()
		var inserted bool
		q := `INSERT INTO captcha.keks (kek_id,kek_data) VALUES ($1,$2) ON CONFLICT (kek_id) DO NOTHING RETURNING TRUE`
		err = tx.QueryRow(q, id, kek).Scan(&inserted)
		if err != nil {
			if err == sql.ErrNoRows {
				// conflict - already exists
				if nfail < 10 {
					nfail++
					continue
				}
			}
			err = p.sqlError("captcha keks insert queryrow scan", err)
			return
		}
		if !inserted {
			// shouldn't happen
			err = p.sqlError("bad inserted value", err)
			return
		}

		// inserted OKAY

		err = tx.Commit()
		if err != nil {
			err = p.sqlError("kek tx commit", err)
			return
		}

		keks = append([]captchastore.KEKInfo{{ID: id, KEK: kek}}, keks...)
		return
	}
}

func (p PSQLStore) sqlError(when string, err error) error {
	return psql.SQLError(p.log, when, err)
}
