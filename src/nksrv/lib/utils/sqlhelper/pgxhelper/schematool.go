package pgxhelper

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"nksrv/lib/utils/sqlbucket"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type schemaAndVer struct {
	s []string
	v int
}

type PGXSchemaTool struct {
	current    []string       // "current" seed
	seeds      []schemaAndVer // various versions seeds
	migrations []schemaAndVer // version upgrades
	maxVer     int            // max ver, either ver of "current" or maximum reachable via seeds and migrations
	versioner  Versioner
}

func NewSchemaTool(dir fs.FS) (_ PGXSchemaTool, err error) {

	var tool PGXSchemaTool

	var objects = make(map[string][]string)
	var currentVer = -1

	err = fs.WalkDir(dir, "schema", func(path string, d fs.DirEntry, xerr error) error {
		if xerr != nil {
			return xerr
		}
		if d.IsDir() {
			if d.Name()[0] == '.' || d.Name()[0] == '_' {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name()[0] == '.' || d.Name()[0] == '_' {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".sql") {
			return fmt.Errorf("%q is not sql file", path)
		}

		q, xerr := sqlbucket.New().
			WithName("x").
			WithNeedSemicolon(true).
			WithNoNext(true).
			LoadFromFS(dir, path)
		if xerr != nil {
			return fmt.Errorf("error loading %q: %w", path, xerr)
		}

		ver := strings.TrimPrefix(path, "schema/")
		i := strings.IndexAny(ver, "./_")
		if i <= 0 {
			return fmt.Errorf("invalid path %q", path)
		}
		ver = ver[:i]

		objects[ver] = append(objects[ver], q["x"][0])

		if ver == "current" && q["version"] != nil {
			if currentVer >= 0 {
				return fmt.Errorf("current version redeclared")
			}
			sv := q["version"][0]
			cv, e := strconv.ParseUint(sv, 10, 32)
			if e != nil || cv > 0x7FffFFff {
				return fmt.Errorf("invalid current version %q", sv)
			}
			currentVer = int(cv)
		}

		return nil
	})
	if err != nil {
		return
	}

	for k, v := range objects {
		if k == "current" {
			tool.current = v
			continue
		}
		if k[0] == 'v' {
			sv := k[1:]
			cv, e := strconv.ParseUint(sv, 10, 32)
			if e != nil || cv > 0x7FffFFff {
				err = fmt.Errorf("invalid v version %q", sv)
				return
			}
			tool.migrations = append(tool.migrations, schemaAndVer{
				s: v,
				v: int(cv),
			})
			continue
		}
		if k[0] == 's' {
			sv := k[1:]
			cv, e := strconv.ParseUint(sv, 10, 32)
			if e != nil || cv > 0x7FffFFff {
				err = fmt.Errorf("invalid s version %q", sv)
				return
			}
			tool.seeds = append(tool.seeds, schemaAndVer{
				s: v,
				v: int(cv),
			})
			continue
		}
		err = fmt.Errorf("unknown item %q", k)
		return
	}

	sort.Slice(tool.migrations, func(i, j int) bool {
		return tool.migrations[i].v < tool.migrations[j].v
	})
	sort.Slice(tool.seeds, func(i, j int) bool {
		return tool.seeds[i].v < tool.seeds[j].v
	})
	for i := 1; i < len(tool.seeds); i++ {
		if tool.seeds[i-1].v == tool.seeds[i].v {
			err = fmt.Errorf(
				"invalid config: duplicate seed entry")
			return
		}
	}
	for i := 1; i < len(tool.migrations); i++ {
		if tool.migrations[i-1].v == tool.migrations[i].v {
			err = fmt.Errorf(
				"invalid config: duplicate migration entry")
			return
		}
	}

	if tool.current != nil {
		if len(tool.seeds) != 0 {
			ms := tool.seeds[len(tool.seeds)-1]
			if ms.v > currentVer {
				err = fmt.Errorf(
					"invalid config: current ver %d < seed ver %d", currentVer, ms.v)
				return
			}
			if ms.v == currentVer && !reflect.DeepEqual(ms.s, tool.current) {
				err = fmt.Errorf(
					"invalid config: current and seed mismatch")
				return
			}
		}
		if len(tool.migrations) != 0 {
			if mmv := tool.migrations[len(tool.migrations)-1].v; mmv > currentVer {
				err = fmt.Errorf(
					"invalid config: current ver %d < migration ver %d", currentVer, mmv)
				return
			}
		}
	}

	tool.maxVer = currentVer
	if len(tool.seeds) != 0 {
		if msv := tool.seeds[len(tool.seeds)-1].v; msv > tool.maxVer {
			tool.maxVer = msv
		}
	}
	if len(tool.migrations) != 0 {
		if mmv := tool.migrations[len(tool.migrations)-1].v; mmv > tool.maxVer {
			tool.maxVer = mmv
		}
	}

	if tool.maxVer < 0 {
		err = errors.New("no seeds, no migrations")
		return
	}

	tool.versioner = TableVersioner{}

	return tool, nil
}

func (tool *PGXSchemaTool) CheckDBConfig(cfg *pgx.ConnConfig, comp string) (err error) {
	conn, err := pgx.ConnectConfig(context.Background(), cfg)
	if err != nil {
		return
	}
	defer func() {
		e := conn.Close(context.Background())
		if err == nil && e != nil {
			err = e
		}
	}()

	err = tool.CheckDBConn(conn, comp)
	return
}

func (tool *PGXSchemaTool) MigrateDBConfig(cfg *pgx.ConnConfig, comp string) (didSomething bool, err error) {
	conn, err := pgx.ConnectConfig(context.Background(), cfg)
	if err != nil {
		return
	}
	defer func() {
		e := conn.Close(context.Background())
		if err == nil && e != nil {
			err = e
		}
	}()

	didSomething, err = tool.MigrateDBConn(conn, comp)
	return
}

type NeedsMigrationError struct{ s string }

func (e NeedsMigrationError) Error() string { return e.s }

var (
	ErrNeedsInitialization = NeedsMigrationError{"database needs initialization"}
	ErrNeedsMigration      = NeedsMigrationError{"database needs migration"}
)

func (tool *PGXSchemaTool) CheckDBConn(conn *pgx.Conn, comp string) error {
	nowVer, err := tool.versioner.GetVersion(conn, comp)
	if err != nil {
		return err
	}
	if nowVer == tool.maxVer {
		// we don't need to migrate
		return nil
	}
	if nowVer < 0 {
		// needs initialization
		return ErrNeedsInitialization
	}
	err = tool.checkCanUpgrade(nowVer)
	if err != nil {
		return err
	}
	// we could do it
	return ErrNeedsMigration
}

var errVersionRace = errors.New("version race")

func (tool *PGXSchemaTool) checkCanUpgrade(nowVer int) error {
	if nowVer > tool.maxVer {
		return fmt.Errorf("database version higher than our (db: v%d, our: v%d)", nowVer, tool.maxVer)
	}
	for _, m := range tool.migrations {
		if nowVer+1 < m.v {
			// migration version too low
			continue
		}
		nowVer++
		// migration version seems either equal or higher
		if nowVer > m.v {
			return fmt.Errorf("database needs update to v%d, but we can't perform it", nowVer)
		}
	}
	if nowVer != tool.maxVer {
		return fmt.Errorf("database needs update to v%d from v%d, but we can't perform it", tool.maxVer, nowVer)
	}
	return nil
}

func (tool *PGXSchemaTool) checkCanSeed() error {
	if tool.current != nil {
		return nil
	}
	if len(tool.seeds) == 0 {
		return fmt.Errorf("cannot initialize database (no seeds)")
	}
	return tool.checkCanUpgrade(tool.seeds[len(tool.seeds)-1].v)
}

func (tool *PGXSchemaTool) MigrateDBConn(conn *pgx.Conn, comp string) (didSomething bool, err error) {

	cRepeat := 0

reVer:
	nowVer, err := tool.versioner.GetVersion(conn, comp)
	if err != nil {
		return
	}

	if nowVer == tool.maxVer {
		// we don't actually need update
		return
	}

	if nowVer >= 0 {
		err = tool.checkCanUpgrade(nowVer)
		if err != nil {
			return
		}
		err = tool.performUpgrade(conn, comp, nowVer)
	} else {
		err = tool.checkCanSeed()
		if err != nil {
			return
		}
		err = tool.performSeed(conn, comp, nowVer)
	}
	if err != nil {
		if err == errVersionRace {
			if cRepeat >= 10 {
				return
			}
			cRepeat++

			goto reVer
		}
		return
	}

	didSomething = true
	return
}

func (tool *PGXSchemaTool) performUpgrade(conn *pgx.Conn, comp string, nowVer int) (err error) {

	tx, err := conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(context.Background())
		}
	}()

	err = tool.versioner.SetVersion(tx, comp, tool.maxVer, nowVer)
	if err != nil {
		return
	}

	c := 1
	for _, m := range tool.migrations {
		if nowVer+c < m.v {
			// migration version too low
			continue
		}
		err = executeMigration(tx, m.s)
		if err != nil {
			return
		}
	}

	err = tx.Commit(context.Background())
	return
}

func (tool *PGXSchemaTool) performSeed(conn *pgx.Conn, comp string, nowVer int) (err error) {

	tx, err := conn.Begin(context.Background())
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback(context.Background())
		}
	}()

	err = tool.versioner.SetVersion(tx, comp, tool.maxVer, nowVer)
	if err != nil {
		return
	}

	if tool.current != nil {
		err = executeMigration(tx, tool.current)
		if err != nil {
			return
		}
	} else {
		s := tool.seeds[len(tool.seeds)-1]

		err = executeMigration(tx, s.s)
		if err != nil {
			return
		}
		nowVer = s.v

		for _, m := range tool.migrations {
			if nowVer+1 < m.v {
				// migration version too low
				continue
			}
			err = executeMigration(tx, m.s)
			if err != nil {
				return
			}
			nowVer++
		}
	}

	err = tx.Commit(context.Background())
	return
}

func executeMigration(tx pgx.Tx, statements []string) error {
	for i, s := range statements {
		if s == "" {
			continue
		}
		_, err := tx.Exec(context.Background(), s, pgx.QuerySimpleProtocol(true))
		if err != nil {

			if xerr, _ := err.(*pgconn.PgError); xerr != nil {

				var pos, ss, se int
				if xerr.Position != 0 || xerr.Line == 0 {
					// character position -> byte position
					for i := range s {
						pos++
						if pos >= int(xerr.Position) {
							pos = i
							break
						}
					}
					// start and end of relevant line
					ss, se = pos, pos
					for ss > 0 && s[ss-1] != '\n' {
						ss--
					}
					for se < len(s) && s[se] != '\n' {
						se++
					}
				} else {
					// position wasn't provided, but line num was
					for i := 0; i < int(xerr.Line); i++ {
						x := strings.IndexByte(s[ss:], '\n')
						if x < 0 {
							ss = len(s)
							break
						}
						ss = x
					}
					pos = ss
					se = ss
					for se < len(s) && s[se] != '\n' {
						se++
					}
				}

				return fmt.Errorf(
					"sql error executing %dth migration part (%w): detail[%s] hint[%s] pos[%d] line[%s] linenum[%d]",
					i, err, xerr.Detail, xerr.Hint, utf8.RuneCountInString(s[ss:pos]), s[ss:se], xerr.Line,
				)
			}
			return fmt.Errorf("error executing %dth migration part: %w", i, err)
		}
	}
	return nil
}

func CheckServerVersion(q pgxQueryRower, verReq int) error {
	var verNow int
	err := q.QueryRow(context.Background(), "SHOW server_version_num", pgx.QuerySimpleProtocol(true)).Scan(&verNow)
	if err != nil {
		return err
	}
	if verNow < verReq {
		return fmt.Errorf("we require at least server version %d, got %d", verReq, verNow)
	}
	return nil
}
