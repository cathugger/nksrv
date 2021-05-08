package initsql

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"

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

func prepareStatementsForConn(ctx context.Context, conn *pgx.Conn, src *[SISize]string) (err error) {
	for i := 0; i < SISize; i++ {
		stn := StatementIndexEntry(i).String()
		_, err = conn.Prepare(ctx, stn, src[i])
		if err != nil {
			return
		}
	}
	return nil
}

/*
func (sp *PSQLIB) prepareStatements() (err error) {
	if sp.StPrep[0] != nil {
		panic("already prepared")
	}
	for i := range StListX {

		s := StListX[i]
		sp.StPrep[i], err = sp.DB.DB.Prepare(s)
		if err != nil {

			if pe, _ := err.(*pq.Error); pe != nil {

				pos, _ := strconv.Atoi(pe.Position)
				ss, se := pos, pos
				for ss > 0 && s[ss-1] != '\n' {
					ss--
				}
				for se < len(s) && s[se] != '\n' {
					se++
				}

				return fmt.Errorf(
					"err preparing %d %q stmt: pos[%s] msg[%s] detail[%s] line[%s]\nstmt:\n%s",
					i, stNames[i].Name, pe.Position, pe.Message, pe.Detail, s[ss:se], s)
			}

			return fmt.Errorf("error preparing %d %q stmt: %v",
				i, stNames[i].Name, err)
		}
	}
	return
}

*/
