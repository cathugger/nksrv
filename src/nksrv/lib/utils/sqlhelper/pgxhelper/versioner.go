package pgxhelper

import (
	"context"
	"errors"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type Versioner interface {
	GetVersion(h *pgx.Conn, comp string) (_ int, err error)
	SetVersion(h pgx.Tx, comp string, ver, oldver int) error
}

type TableVersioner struct{}

var errEmptyComponent = errors.New("empty component not allowed")

func isNoTableError(e error) bool {
	var pgErr *pgconn.PgError
	return errors.As(e, &pgErr) && pgErr != nil && pgErr.Code == "42P01"
}

func (TableVersioner) GetVersion(h *pgx.Conn, comp string) (_ int, err error) {

	if comp == "" {
		return -1, errEmptyComponent
	}

	q := `
SELECT
	version
FROM
	public.components_versions
WHERE
	component = $1
`

	var ver int
	err = h.QueryRow(context.Background(), q, pgx.QuerySimpleProtocol(true), comp).Scan(&ver)
	if err != nil && isNoTableError(err) {
		// run init logic and retry
		err = initTableVersioner(h)
		if err == nil {
			err = h.QueryRow(context.Background(), q, pgx.QuerySimpleProtocol(true), comp).Scan(&ver)
		}
	}
	if err != nil {
		if err == pgx.ErrNoRows {
			return -1, nil
		}
		if isNoTableError(err) {
			return -1, nil
		}
		return -1, err
	}

	return ver, nil
}

func initTableVersioner(h *pgx.Conn) error {
	q := `
CREATE TABLE IF NOT EXISTS public.components_versions (
	component  TEXT     NOT NULL PRIMARY KEY,
	version    INTEGER  NOT NULL
)
`
	_, err := h.Exec(context.Background(), q, pgx.QuerySimpleProtocol(true))
	return err
}

func (TableVersioner) SetVersion(h pgx.Tx, comp string, ver, oldver int) error {

	if comp == "" {
		return errEmptyComponent
	}

	if oldver >= 0 {

		q := `
UPDATE
	public.components_versions
SET
	version = $2
WHERE
	component = $1 AND version = $3
RETURNING
	version
`

		var dummy int
		err := h.QueryRow(
			context.Background(), q,
			pgx.QuerySimpleProtocol(true),
			comp, ver, oldver,
		).Scan(&dummy)
		if err != nil {
			if err == pgx.ErrNoRows {
				return errVersionRace
			}
			return err
		}

	} else {

		q := `
INSERT INTO
	public.components_versions (
		component,
		version
	)
	VALUES (
		$1,
		$2
	)
ON CONFLICT
	DO NOTHING
RETURNING
	version
`

		var dummy int
		err := h.QueryRow(
			context.Background(), q,
			pgx.QuerySimpleProtocol(true),
			comp, ver,
		).Scan(&dummy)
		if err != nil {
			if err == pgx.ErrNoRows {
				return errVersionRace
			}
			return err
		}
	}

	return nil
}
