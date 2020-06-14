package psqlib

func (dbib *PSQLIB) InitAndPrepare() (err error) {
	valid, err := dbib.CheckDB()
	if err != nil {
		return fmt.Errorf("error checking: %v", err)
	}
	if !valid {
		dbib.Log.LogPrint(NOTICE,
			"uninitialized db, attempting to initialize")

		err = dbib.InitDB()
		if err != nil {
			return fmt.Errorf("error initializing: %v", err)
		}

		valid, err = dbib.CheckDB()
		if err != nil {
			return fmt.Errorf("error checking (2): %v", err)
		}
		if !valid {
			return errors.New("database still not valid after initialization")
		}
	}

	err = dbib.Prepare()
	if err != nil {
		return
	}

	return
}

func NewInitAndPrepare(cfg Config) (db *PSQLIB, err error) {
	db, err = NewPSQLIB(cfg)
	if err != nil {
		return
	}

	err = db.InitAndPrepare()
	if err != nil {
		return
	}

	return
}
