package psqlib

import (
	"database/sql"
	"fmt"
	"io"

	"nksrv/lib/date"
	"nksrv/lib/nntp"
)

type PullerDB struct {
	sp    *PSQLIB
	id    int64
	nonce int64

	ngp_thispuller newGroupPolicy
	notrace        bool

	temp_rows *sql.Rows
}

var _ nntp.PullerDatabase = (*PullerDB)(nil)

func (s *PullerDB) autoAddGroup(group string) bool {
	return s.sp.ngp_global.checkGroup(group) ||
		s.sp.ngp_anypuller.checkGroup(group) ||
		s.ngp_thispuller.checkGroup(group)
}

func (s *PullerDB) getNonce() int64 {
	// not to be used in multithreaded context
	if s.nonce == 0 {
		s.nonce = date.NowTimeUnixMilli()
		if s.nonce == 0 {
			s.nonce = 1
		}
	}
	return s.nonce
}
func (s *PullerDB) nextNonce() int64 {
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

func (s *PullerDB) GetLastNewNews() (t int64, err error) {
	e := s.sp.st_prep[st_puller_get_last_newnews].
		QueryRow(s.id).
		Scan(&t)
	if e != nil {
		if e == sql.ErrNoRows {
			return 0, nil
		}
		return 0, s.sp.sqlError("puller_get_last_newnews query scan", e)
	}
	return
}
func (s *PullerDB) UpdateLastNewNews(t int64) error {
	_, e := s.sp.st_prep[st_puller_set_last_newnews].
		Exec(s.id, t)
	if e != nil {
		return s.sp.sqlError("puller_set_last_newnews query execution", e)
	}
	return nil
}

func (s *PullerDB) GetLastNewGroups() (t int64, err error) {
	e := s.sp.st_prep[st_puller_get_last_newsgroups].
		QueryRow(s.id).
		Scan(&t)
	if e != nil {
		if e == sql.ErrNoRows {
			return 0, nil
		}
		return 0, s.sp.sqlError("puller_get_last_newsgroups query scan", e)
	}
	return
}
func (s *PullerDB) UpdateLastNewGroups(t int64) error {
	_, e := s.sp.st_prep[st_puller_set_last_newsgroups].
		Exec(s.id, t)
	if e != nil {
		return s.sp.sqlError("puller_set_last_newsgroups query execution", e)
	}
	return nil
}

func (s *PullerDB) GetGroupID(group []byte) (int64, error) {
	unsafe_sgroup := unsafeBytesToStr(group)

	loopn := 0
	for {
		var bid boardID
		var lid sql.NullInt64

		e := s.sp.st_prep[st_puller_get_group_id].
			QueryRow(s.id, unsafe_sgroup).
			Scan(&bid, &lid)
		if e != nil {
			if e == sql.ErrNoRows {
				if !s.autoAddGroup(unsafe_sgroup) || loopn >= 20 {
					return -1, nil
				}
				// proceed with adding
			} else {
				// SQL error
				return -1, s.sp.sqlError("puller_get_group_id query scan", e)
			}
		} else {
			// found something. board exists but ID may be NULL. which is OK
			return lid.Int64, nil
		}

		loopn++

		// if we're here, then we need to make new board
		bi := s.sp.IBDefaultBoardInfo()
		bi.Name = unsafe_sgroup
		e, dup := s.sp.addNewBoard(bi)
		if e != nil && !dup {
			return -1, fmt.Errorf("addNewBoard error: %v", e)
		}
	}
}
func (s *PullerDB) UpdateGroupID(group string, id uint64) error {
	var es string
	var e error
	if id != 0 {
		es = "puller_set_group_id query execution"
		_, e = s.sp.st_prep[st_puller_set_group_id].
			Exec(s.id, group, id)
	} else {
		es = "puller_unset_group_id query execution"
		_, e = s.sp.st_prep[st_puller_unset_group_id].
			Exec(s.id, group)
	}
	if e != nil {
		return s.sp.sqlError(es, e)
	}
	return nil
}

func (s *PullerDB) StartTempGroups() error {
	s.nextNonce()
	return nil
}
func (s *PullerDB) CancelTempGroups() error {
	// nothing :^)
	return nil
}
func (s *PullerDB) FinishTempGroups(partial bool) (err error) {
	// there, if partial == false, we can clean up old
	if !partial {
		nonce := s.getNonce()
		q := `DELETE FROM ib0.puller_group_track
WHERE sid = $1 AND last_use <> $2`
		_, err = s.sp.db.DB.Exec(q, s.id, nonce)
		if err != nil {
			return s.sp.sqlError("puller_group_track delete query execution", err)
		}
	}
	return
}
func (s *PullerDB) DoneTempGroups() error {
	// we would clean something up if we cared
	return nil
}
func (s *PullerDB) StoreTempGroupID(
	group []byte, new_id uint64) error {

	q := `INSERT INTO ib0.puller_group_track AS sgt (sid,bid,last_use,last_max,next_max)
SELECT $1 AS sid, b_id AS bid, $3, 0, $4
FROM ib0.boards xb
WHERE b_name=$2
ON CONFLICT (sid,bid)
	DO UPDATE SET last_use=$3, next_max=$4
	WHERE sgt.sid=EXCLUDED.sid AND sgt.bid=EXCLUDED.bid`
	nonce := s.getNonce()
	_, e := s.sp.db.DB.Exec(q, s.id, group, nonce, new_id)
	if e != nil {
		return s.sp.sqlError("puller_group_track upsert query execution", e)
	}
	return nil
}
func (s *PullerDB) StoreTempGroup(group []byte) error {
	q := `INSERT INTO ib0.puller_group_track AS sgt (sid,bid,last_use,last_max,next_max)
SELECT $1 AS sid, b_id AS bid, $3, 0, -1
FROM ib0.boards xb
WHERE b_name=$2
ON CONFLICT (sid,bid)
	DO UPDATE SET last_use=$3, next_max=-1
	WHERE sgt.sid=EXCLUDED.sid AND sgt.bid=EXCLUDED.bid`
	nonce := s.getNonce()
	_, e := s.sp.db.DB.Exec(q, s.id, group, nonce)
	if e != nil {
		return s.sp.sqlError("puller_group_track upsert query execution", e)
	}
	return nil
}
func (s *PullerDB) LoadTempGroup() (
	group string, new_id int64, old_id uint64, err error) {

	// TODO throw out this deadlock-inducing logic
	if s.temp_rows == nil {
		s.temp_rows, err = s.sp.st_prep[st_puller_load_temp_groups].
			Query(s.id, s.nonce)
		if err != nil {
			s.temp_rows = nil
			err = s.sp.sqlError("puller_group_track load query", err)
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
		err = s.sp.sqlError("puller_group_track load query scan", err)
		return
	}
	return
}

func (s *PullerDB) IsArticleWanted(msgid FullMsgIDStr) (bool, error) {
	cmsgid := cutMsgID(msgid)
	// check if we already have it
	exists, err := s.sp.nntpCheckArticleExistsOrBanned(cmsgid)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (s *PullerDB) DoesReferenceExist(
	ref FullMsgIDStr) (exists bool, err error) {

	exists, _, err = s.sp.nntpCheckArticleValidAndBanned(cutMsgID(ref))
	return
}

var (
	nntpPullerDir = "_sin"
)

func (s *PullerDB) ReadArticle(
	r io.Reader, msgid CoreMsgIDStr, expectgroup string) (
	err error, unexpected bool, wantroot FullMsgIDStr) {

	info, newname, H, err, unexpected, wantroot :=
		s.sp.handleIncoming(r, msgid, expectgroup, nntpPullerDir, s.notrace)
	if err != nil {
		return
	}

	s.sp.nntpSendIncomingArticle(newname, H, info)
	return
}

func (sp *PSQLIB) getPullerNonce() int64 {
	// not to be used in multithreaded context
	if sp.puller_nonce == 0 {
		sp.puller_nonce = date.NowTimeUnixMilli()
		if sp.puller_nonce == 0 {
			sp.puller_nonce = 1
		}
	}
	return sp.puller_nonce
}

func (sp *PSQLIB) NewPullerDB(name string, autoadd string, notrace bool) (*PullerDB, error) {
	q := `INSERT INTO ib0.puller_list AS sl (sname,last_use)
VALUES ($1,$2)
ON CONFLICT (sname)
DO
	UPDATE SET last_use = $2
	WHERE sl.sname = $1
RETURNING sid`
	nonce := sp.getPullerNonce()
	ngp, e := makeNewGroupPolicy(autoadd)
	if e != nil {
		return nil, e
	}
	db := &PullerDB{
		sp:             sp,
		ngp_thispuller: ngp,
		notrace:        notrace,
	}
	e = sp.db.DB.
		QueryRow(q, name, nonce).
		Scan(&db.id)
	if e != nil {
		return nil, sp.sqlError("puller_list upsert query scan", e)
	}
	return db, nil
}

func (sp *PSQLIB) ClearPullerDBs() error {
	nonce := sp.getPullerNonce()
	q := `DELETE FROM ib0.puller_list WHERE last_use <> $1`
	_, e := sp.db.DB.Exec(q, nonce)
	if e != nil {
		return sp.sqlError("puller_list delete query execution", e)
	}
	return nil
}
