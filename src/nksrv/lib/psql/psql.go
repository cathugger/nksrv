package psql

// not-too-generic PSQL connector
// can be used by more concrete forum packages

import (
	"errors"
	"fmt"
	"time"

	. "nksrv/lib/logx"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Config struct {
	ConnDriver      string
	ConnStr         string
	ConnMaxLifetime float64
	MaxIdleConns    int32
	MaxOpenConns    int32
	Logger          LoggerX
}

var DefaultConfig = Config{
	ConnDriver:      "postgres",
	ConnStr:         "",
	ConnMaxLifetime: 0.0,
	MaxIdleConns:    0,
	MaxOpenConns:    0,
}

type PSQL struct {
	DB  *sqlx.DB
	log Logger
	id  string
}

func OpenPSQL(cfg Config) (PSQL, error) {
	db, err := sqlx.Open(cfg.ConnDriver, cfg.ConnStr)
	if err != nil {
		return PSQL{}, err
	}

	if cfg.ConnMaxLifetime > 0.0 {
		db.SetConnMaxLifetime(time.Duration(float64(time.Second) *
			cfg.ConnMaxLifetime))
	}

	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(int(cfg.MaxIdleConns))
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(int(cfg.MaxOpenConns))
	}

	p := PSQL{DB: db}
	p.id = fmt.Sprintf("psqlib.%p", p.DB)
	p.log = NewLogToX(cfg.Logger, p.id)

	return p, nil
}

func (p PSQL) Close() error {
	return p.DB.Close()
}

func (p PSQL) ID() string {
	return p.id
}

func OpenAndPrepare(cfg Config) (db PSQL, err error) {
	db, err = OpenPSQL(cfg)
	if err != nil {
		err = fmt.Errorf("error opening: %v", err)
		return
	}
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	valid, err := db.IsValidDB()
	if err != nil {
		err = fmt.Errorf("error validating: %v", err)
		return
	}
	// if not valid, try to create
	if !valid {
		db.log.LogPrint(NOTICE, "uninitialized PSQL db, attempting to initialize")

		db.InitDB()

		// revalidate
		valid, err = db.IsValidDB()
		if err != nil {
			err = fmt.Errorf("error validating (2): %v", err)
			return
		}
		if !valid {
			err = errors.New("database still not valid after initialization")
			return
		}
	}

	err = db.CheckVersion()
	if err != nil {
		err = fmt.Errorf("version check fail: %v", err)
		return
	}

	return
}
