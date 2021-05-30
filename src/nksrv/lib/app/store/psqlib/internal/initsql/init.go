package initsql

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"

	. "nksrv/lib/app/store/psqlib/internal/basesql"
	"nksrv/lib/utils/sqlbucket"
)

func loadStatementsFromFS(src fs.FS) (_ sqlbucket.Bucket, err error) {
	list, err := fs.ReadDir(src, "statements")
	if err != nil {
		return
	}
	var dst sqlbucket.Bucket
	for _, e := range list {
		name := e.Name()
		if len(name) == 0 || name[0] == '.' || name[0] == '_' || !strings.HasSuffix(name, ".sql") || e.IsDir() {
			continue
		}
		dst, err = sqlbucket.New().
			WithNeedSemicolon(true).
			WithNoNext(true).
			WithBase(dst).
			LoadFromFS(src, path.Join("statements", name))
		if err != nil {
			return
		}
	}
	return dst, nil
}

func compileStatementList(src sqlbucket.Bucket) (_ *[SISize]string, err error) {
	dst := new([SISize]string)
	for i := 0; i < SISize; i++ {
		stn := StatementIndexEntry(i).String()
		st := src[stn]
		if len(st) == 0 {
			err = fmt.Errorf("%q statement err: not found", stn)
			return
		}
		if len(st) > 1 {
			err = fmt.Errorf("%q statement err: multiple statements", stn)
			return
		}
		delete(src, stn)
		dst[i] = st[0]
	}
	if len(src) != 0 {
		err = fmt.Errorf("%d unprocessed statements left", len(src))
		return
	}
	return dst, nil
}

func LoadStatementsFromFS(src fs.FS) (_ *[SISize]string, err error) {
	bucket, err := loadStatementsFromFS(src)
	if err != nil {
		err = fmt.Errorf("error loading statements: %w", err)
		return
	}
	list, err := compileStatementList(bucket)
	if err != nil {
		err = fmt.Errorf("error compiling statement list: %w", err)
		return
	}
	return list, nil
}

func PrepareStatementsForConn(ctx context.Context, conn *pgx.Conn, src *[SISize]string) (err error) {
	for i := 0; i < SISize; i++ {
		stn := StatementIndexEntry(i).String()
		s := src[i]
		_, err = conn.Prepare(ctx, stn, s)
		if err != nil {
			if xerr, _ := err.(*pgconn.PgError); xerr != nil {
				ss, se := xerr.Position, xerr.Position
				for ss > 0 && s[ss-1] != '\n' {
					ss--
				}
				for int(se) < len(s) && s[se] != '\n' {
					se++
				}
				err = fmt.Errorf(
					"err preparing %d %q stmt (%w): pos[%d] msg[%s] detail[%s] line[%s]\nstmt:\n%s",
					i, stn, err, xerr.Position, xerr.Message, xerr.Detail, s[ss:se], s,
				)
			} else {
				err = fmt.Errorf(
					"weird err preparing %d %q stmt: %w",
					i, stn, err,
				)
			}
			return
		}
	}
	return nil
}
