package psqlib

import (
	"database/sql"
	"io"

	"nekochan/lib/date"
	"nekochan/lib/nntp"
)

type ScraperDB struct {
	sp    *PSQLIB
	id    int32
	nonce int32

	temp_rows *sql.Rows
}

var _ nntp.ClientDatabase = (*ScraperDB)(nil)

func (s *ScraperDB) getNonce() int32 {
	// not to be used in multithreaded context
	if s.nonce == 0 {
		nowtime := date.NowTimeUnix()
		s.nonce = int32(nowtime)
		if s.nonce == 0 {
			s.nonce = 1
		}
	}
	return s.nonce
}
func (s *ScraperDB) nextNonce() int32 {
	if s.nonce != 0 {
		s.nonce++
		if s.nonce == 0 {
			s.nonce = 1
		}
	} else {
		s.getNonce()
	}
	return s.nonce
}

func (s *ScraperDB) GetLastNewNews() (t int64, err error) {
	q := `SELECT last_newnews FROM ib0.scraper_last_newnews WHERE sid=$1`
	e := s.sp.db.DB.
		QueryRow(q, s.id).
		Scan(&t)
	if e != nil {
		if e == sql.ErrNoRows {
			return 0, nil
		}
		return 0, s.sp.sqlError("scraper_last_newnews query scan", e)
	}
	return
}
func (s *ScraperDB) UpdateLastNewNews(t int64) error {
	q := `INSERT INTO ib0.scraper_last_newnews (sid,last_newnews)
VALUES ($1,$2)
ON CONFLICT DO UPDATE SET last_newnews = $2
WHERE sid = $1`
	_, e := s.sp.db.DB.Exec(q, s.id, t)
	if e != nil {
		return s.sp.sqlError("scraper_last_newnews upsert query execution", e)
	}
	return nil
}

func (s *ScraperDB) GetLastNewGroups() (t int64, err error) {
	q := `SELECT last_newgroups FROM ib0.scraper_last_newgroups WHERE sid=$1`
	e := s.sp.db.DB.
		QueryRow(q, s.id).
		Scan(&t)
	if e != nil {
		if e == sql.ErrNoRows {
			return 0, nil
		}
		return 0, s.sp.sqlError("scraper_last_newgroups query scan", e)
	}
	return
}
func (s *ScraperDB) UpdateLastNewGroups(t int64) error {
	q := `INSERT INTO ib0.scraper_last_newgroups (sid,last_newgroups)
VALUES ($1,$2)
ON CONFLICT DO UPDATE SET last_newgroups = $2
WHERE sid = $1`
	_, e := s.sp.db.DB.Exec(q, s.id, t)
	if e != nil {
		return s.sp.sqlError("scraper_last_newgroups upsert query execution", e)
	}
	return nil
}

func (s *ScraperDB) GetGroupID(group []byte) (int64, error) {
	q := `WITH
	sg AS (
		SELECT bid FROM ib0.boards WHERE bname = $2 LIMIT 1
	),
	st AS (
		SELECT xt.bid AS bid, xt.last_max AS last_max
		FROM ib0.scraper_group_track xt
		JOIN sg
		ON sg.bid = xt.bid
		WHERE xt.sid = $1
	)
SELECT sg.bid,st.last_max
FROM sg
LEFT JOIN st
ON sg.bid = st.bid`
	var bid boardID
	var lid sql.NullInt64

	e := s.sp.db.DB.
		QueryRow(q, s.id, group).
		Scan(&bid, &lid)
	if e != nil {
		if e == sql.ErrNoRows {
			return -1, nil
		}
		return -1, s.sp.sqlError("GetGroupID query scan", e)
	}
	return lid.Int64, nil
}
func (s *ScraperDB) UpdateGroupID(group []byte, id uint64) error {
	q := `UPDATE ib0.scraper_group_track AS st
SET last_max = $3
FROM ib0.boards AS xb
WHERE st.sid=$1 AND xb.bname=$2 AND st.bid=xb.bid`
	_, e := s.sp.db.DB.Exec(q, s.id, group, id)
	if e != nil {
		return s.sp.sqlError("scraper_group_track update query execution", e)
	}
	return nil
}

func (s *ScraperDB) StartTempGroups() error {
	s.nextNonce()
	return nil
}
func (s *ScraperDB) CancelTempGroups() {
	// nothing :^)
}
func (s *ScraperDB) FinishTempGroups(partial bool) {
	// there, if partial == false, we can clean up old
	if !partial {
		nonce := s.getNonce()
		q := `DELETE FROM ib0.scraper_group_track
WHERE sid = $1 AND last_use <> $2`
		_, e := s.sp.db.DB.Exec(q, s.id, nonce)
		if e != nil {
			s.sp.sqlError("scraper_group_track delete query execution", e)
		}
	}
}
func (s *ScraperDB) DoneTempGroups() {
	// we would clean something up if we cared
}
func (s *ScraperDB) StoreTempGroupID(
	group []byte, new_id uint64, old_id uint64) error {

	q := `INSERT INTO ib0.scraper_group_track (sid,bid,last_use,last_max,next_max)
SELECT $1 AS sid, bid, $3, 0, $4
FROM ib0.boards xb
WHERE bname=$2
ON CONFLICT
	DO UPDATE SET last_use=$3, next_max=$4
	WHERE sid=EXCLUDED.sid AND bid=EXCLUDED.bid`
	nonce := s.getNonce()
	_, e := s.sp.db.DB.Exec(q, s.id, group, nonce, new_id)
	if e != nil {
		s.sp.sqlError("scraper_group_track upsert query execution", e)
	}
	return nil
}
func (s *ScraperDB) StoreTempGroup(group []byte, old_id uint64) error {
	q := `INSERT INTO ib0.scraper_group_track (sid,bid,last_use,last_max,next_max)
SELECT $1 AS sid, bid, $3, 0, -1
FROM ib0.boards xb
WHERE bname=$2
ON CONFLICT
	DO UPDATE SET last_use=$3, next_max=-1
	WHERE sid=EXCLUDED.sid AND bid=EXCLUDED.bid`
	nonce := s.getNonce()
	_, e := s.sp.db.DB.Exec(q, s.id, group, nonce)
	if e != nil {
		s.sp.sqlError("scraper_group_track upsert query execution", e)
	}
	return nil
}
func (s *ScraperDB) LoadTempGroup() (
	group string, new_id int64, old_id uint64, err error) {

	if s.temp_rows == nil {
		q := `SELECT xb.bname,xs.next_max,xs.last_max
FROM ib0.scraper_group_track xs
JOIN ib0.boards xb
ON xs.bid = xb.bid
WHERE xs.sid=$1 AND xs.last_use=$2`
		s.temp_rows, err = s.sp.db.DB.Query(q, s.id, s.nonce)
		if err != nil {
			s.temp_rows = nil
			s.sp.sqlError("scraper_group_track load query", err)
			return
		}
	}

	defer func() {
		if err != nil {
			s.temp_rows.Close()
			s.temp_rows = nil
		}
	}()
	if !s.temp_rows.Next() {
		err = s.temp_rows.Err()
		if err == nil {
			err = io.EOF
		}
		return
	}
	err = s.temp_rows.Scan(&group, &new_id, &old_id)
	if err != nil {
		err = s.sp.sqlError("scraper_group_track load query scan", err)
		return
	}
	return
}

func (sp *PSQLIB) getScraperNonce() int32 {
	// not to be used in multithreaded context
	if sp.scraper_nonce == 0 {
		nowtime := date.NowTimeUnix()
		sp.scraper_nonce = int32(nowtime)
		if sp.scraper_nonce == 0 {
			sp.scraper_nonce = 1
		}
	}
	return sp.scraper_nonce
}

func (sp *PSQLIB) NewScraperDB(name string) (*ScraperDB, error) {
	q := `INSERT INTO ib0.scraper_list (sname,last_use)
	VALUES ($1,$2)
	ON CONFLICT DO UPDATE SET last_use = $2
	WHERE sname = $1
	RETURNING sid`
	nonce := sp.getScraperNonce()
	db := &ScraperDB{sp: sp}
	e := sp.db.DB.
		QueryRow(q, name, nonce).
		Scan(&db.id)
	if e != nil {
		return nil, sp.sqlError("scraper_list upsert query scan", e)
	}
	return db, nil
}

func (sp *PSQLIB) ClearScraperDBs() error {
	nonce := sp.getScraperNonce()
	q := `DELETE FROM ib0.scraper_list WHERE last_use <> $1`
	_, e := sp.db.DB.Exec(q, nonce)
	if e != nil {
		return sp.sqlError("scraper_list delete query execution", e)
	}
	return nil
}
