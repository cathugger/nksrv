package pgxhelper

import (
	"context"

	"github.com/jackc/pgx/v4"
)

type pgxQueryer interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}
