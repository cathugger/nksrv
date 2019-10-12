package psql

// not-too-generic PSQL connector
// can be used by more concrete forum packages

import (
	"errors"
	"fmt"
	"sync"
	"time"

	. "nksrv/lib/logx"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Config struct {
	ConnStr         string
	ConnMaxLifetime float64
	MaxIdleConns    int32
	MaxOpenConns    int32
	Logger          LoggerX
}

var DefaultConfig = Config{
	ConnStr:         "",
	ConnMaxLifetime: 0.0,
	MaxIdleConns:    0,
	MaxOpenConns:    0,
}

type PSQL struct {
	DB *sqlx.DB

	connstr string
	lii     sync.Once
	li      *pq.Listener
	limMu   sync.Mutex // only single consumer so RWMutex would be useless
	lim     map[string]ListenCB
	log     Logger
	id      string
}

func OpenPSQL(cfg Config) (PSQL, error) {
	db, err := sqlx.Open("postgres", cfg.ConnStr)
	if err != nil {
		return PSQL{}, err
	}

	if cfg.ConnMaxLifetime > 0.0 {
		db.SetConnMaxLifetime(
			time.Duration(float64(time.Second) *
				cfg.ConnMaxLifetime))
	}

	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(int(cfg.MaxIdleConns))
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(int(cfg.MaxOpenConns))
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return PSQL{}, err
	}

	p := PSQL{DB: db}
	p.id = fmt.Sprintf("psqlib.%p", p.DB)
	p.log = NewLogToX(cfg.Logger, p.id)
	p.connstr = cfg.ConnStr

	return p, nil
}

func (p *PSQL) Close() error {
	e := p.DB.Close()

	// incase didn't exec yet, prevents
	// incase exec'ing rn, waits till done
	p.lii.Do(func() {})

	if p.li != nil {
		p.li.Close()
	}

	return e
}

func (p *PSQL) ID() string {
	return p.id
}

func OpenAndPrepare(cfg Config) (db PSQL, err error) {
	db, err = OpenPSQL(cfg)
	if err != nil {
		err = fmt.Errorf("error opening: %v", err)
		return
	}
	defer func() {
		// XXX won't catch panic
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
