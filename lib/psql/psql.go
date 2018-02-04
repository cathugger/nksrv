package psql

// generic PSQL connector
// can be used by more concrete forum packages

import (
	"../psqlcfg"
	"github.com/jmoiron/sqlx"
	"time"
)

type PSQL struct {
	DB *sqlx.DB
}

func OpenPSQL(cfg psqlcfg.ConfigPSQL) (PSQL, error) {
	db, err := sqlx.Open("postgres", cfg.ConnStr)
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
	return PSQL{DB: db}, nil
}
