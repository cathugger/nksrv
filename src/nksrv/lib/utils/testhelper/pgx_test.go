package testhelper

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4"
)

func TestPGX(t *testing.T) {
	pp, err := NewPGXProvider()
	if err != nil {
		t.Errorf("NewPGXProvider err: %v\n", err)
		return
	}
	defer func() {
		e := pp.Close()
		if e != nil {
			t.Errorf("PGXProvider.Close err: %v\n", e)
		}
	}()

	db, err := pp.NewDatabase()
	if err != nil {
		t.Errorf("pgxProv.NewDatabase err: %v", err)
		return
	}
	defer func() {
		e := db.Close()
		if e != nil {
			t.Errorf("db.Close err: %v", e)
		}
	}()

	conn, err := pgx.ConnectConfig(context.Background(), db.Config)
	if err != nil {
		t.Errorf("pgx.ConnectConfig err: %v", err)
		return
	}
	defer func() {
		e := conn.Close(context.Background())
		if e != nil {
			t.Errorf("conn.Close err: %v", e)
		}
	}()

	var dst int
	err = conn.QueryRow(context.Background(), "SELECT 1").Scan(&dst)
	if err != nil {
		t.Errorf("conn.QueryRow err: %v", err)
		return
	}
	if dst != 1 {
		t.Errorf("conn.QueryRow bogus ret: %d", dst)
		return
	}
}
