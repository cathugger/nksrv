package pgxhelper

import (
	"context"

	"github.com/jackc/pgx/v4"
)

type pgxQueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}
