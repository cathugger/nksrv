package pimod

import (
	"database/sql"

	"nksrv/lib/app/psqlib/internal/pibase"
)

// mod cmd transaction context
type modCtx struct {
	sp *pibase.PSQLIB
	tx *sql.Tx

	ModID       int64
	DelInited   bool
	DelOurModID bool
}
