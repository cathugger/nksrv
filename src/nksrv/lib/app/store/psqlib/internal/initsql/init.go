package initsql

import (
	"fmt"
	"strconv"

	"github.com/lib/pq"

	. "nksrv/lib/app/store/psqlib/internal/basesql"
	"nksrv/lib/utils/sqlbucket"
)

var StListX [StCount]string
var StLoadErr error


func LoadStatements() {
	if StListX[0] != "" {
		panic("already loaded")
	}
	bm := make(map[string]sqlbucket.Bucket)
	for i := 0; i < StCount; i++ {
		sn := StatementIndexEntry(i).String()
		if bm[sn.Bucket] == nil {
			fn := "etc/psqlib/" + sn.Bucket + ".sql"
			sqlbucket.New().LoadFromFS()
			stmts, err := sqlbucket.LoadFromFile(fn)
			if err != nil {
				StLoadErr = fmt.Errorf("err loading %s: %v", fn, err)
				return
			}
			bm[sn.Bucket] = stmts
		}
		sm := bm[sn.Bucket]
		sl := sm[sn.Name]
		if len(sl) != 1 {
			StLoadErr = fmt.Errorf(
				"wrong count %d for statement %s", len(sl), sn)
			return
		}
		StListX[i] = sl[0] + "\n"
	}
}

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

func (sp *PSQLIB) closeStatements() (err error) {
	for i := range StListX {
		if sp.StPrep[i] != nil {
			ex := sp.StPrep[i].Close()
			if err == nil {
				err = ex
			}
			sp.StPrep[i] = nil
		}
	}
	return
}
