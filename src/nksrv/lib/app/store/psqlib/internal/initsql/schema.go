package initsql

import (
	"github.com/jackc/pgx/v4"

	"nksrv/lib/utils/sqlhelper/pgxhelper"
)

func CheckDBConn(tool *pgxhelper.PGXSchemaTool, conn *pgx.Conn) error {
	return tool.CheckDBConn(conn, "ib")
}

func MigrateDBConn(tool *pgxhelper.PGXSchemaTool, conn *pgx.Conn) (didSomething bool, err error) {
	return tool.MigrateDBConn(conn, "ib")
}
