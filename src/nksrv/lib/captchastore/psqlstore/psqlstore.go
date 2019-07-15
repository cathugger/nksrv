package psqlstore

type PSQLStore struct {
	db *psql.PSQL
}


func (sp *PSQLStore) doDbInit() (err error) {
	stmts := [...]string{
		`CREATE SCHEMA captcha`,
		`CREATE TABLE captcha.keks (
	kek_id INTEGER GENERATED ALWAYS AS IDENTITY,
	kek_data BYTEA NOT NULL,

	PRIMARY KEY (kek_id)
)`,
		`CREATE TABLE captcha.solved (
	solved_key BYTEA NOT NULL,
	solved_exp TIMESTAMP WITH TIME ZONE,

	PRIMARY KEY (solved_key)
)`
	}

	tx, err := sp.db.DB.BeginTx(context.Background(), &sql.TxOptions{
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

	for i, s := range stmts["init"] {
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

func (sp *PSQLIB) InitIb0() {
	e := sp.doDbIbit()
	if e != nil {
		panic(e)
	}
}

func (sp *PSQLIB) CheckIb0() (initialised bool, versionerror error) {
	q := "SHOW server_version_num"
	var vernum int64
	err := sp.db.DB.QueryRow(q).Scan(&vernum)
	if err != nil {
		return false, sp.sqlError("server version query", err)
	}
	const verreq = 100000
	if vernum < verreq {
		return false, fmt.Errorf("we require at least server version %d, got %d", verreq, vernum)
	}

	q = "SELECT version FROM capabilities WHERE component = 'ib0' LIMIT 1"
	var ver string
	err = sp.db.DB.QueryRow(q).Scan(&ver)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sp.sqlError("version row query", err)
	}

	if ver != currIb0Version {
		return true, fmt.Errorf("incorrect ib0 schema version: %q (our: %q)", ver, currIb0Version)
	}

	return true, nil
}




func NewPSQLStore(db *psql.PSQL) captchastore.CaptchaStore {
	s := psqlStore{db: db}
	return s
}

func (s *psqlStore) StoreSolved(
	obj []byte, expires, nowtime int64) (fresh bool, err error) {


}

func (ms *MemStore) LoadKEKs(
	ifempty func() (id uint64, kek []byte)) (
	keks []captchastore.KEKInfo, err error) {

	ms.lock.Lock()
	defer ms.lock.Unlock()

	if len(ms.keks) == 0 && ifempty != nil {
		id, kek := ifempty()
		ms.keks = append(ms.keks, captchastore.KEKInfo{ID: id, KEK: kek})
	}
	return ms.keks, nil
}
