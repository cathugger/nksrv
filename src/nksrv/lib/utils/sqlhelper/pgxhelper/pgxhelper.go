package pgxhelper

import (
	"context"
	"fmt"
	"io/fs"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"nksrv/lib/utils/sqlbucket"

	"github.com/jackc/pgx/v4"
)

func DoStuffConfig(cfg *pgx.ConnConfig, comp string, dir fs.FS) (err error) {
	conn, err := pgx.ConnectConfig(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer func() {
		e := conn.Close(context.Background())
		if err == nil && e != nil {
			err = e
		}
	}()

	err = DoStuffConn(conn, comp, dir)
	return
}

type schemaAndVer struct {
	s []string
	v int
}

type PGXSchemaTool struct {
	current    []string // "current" seed
	seeds      []schemaAndVer // various versions seeds
	migrations []schemaAndVer // version upgrades
	maxVer     int // max ver, either ver of "current" or maximum reachable via seeds and migrations
}

func NewSchemaTool(dir fs.FS) (_ PGXSchemaTool, err error) {

	var tool PGXSchemaTool

	var objects = make(map[string][]string)
	var currentVer = -1

	e := fs.WalkDir(dir, "schema", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
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

		q, err := sqlbucket.New().
			WithName("x").
			WithNeedSemicolon(true).
			WithNoNext(true).
			LoadFromFS(dir, path)
		if err != nil {
			return err
		}

		ver := strings.TrimPrefix(path, "schema/")
		i := strings.IndexAny(ver, "./")
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
			cv, err := strconv.ParseUint(sv, 10, 32)
			if err != nil || cv > 0x7FffFFff {
				return fmt.Errorf("invalid current version %q", sv)
			}
			currentVer = int(cv)
		}

		return nil
	})
	if e != nil {
		return e
	}


	for k, v := range objects {
		if k == "current" {
			tool.current = v
			continue
		}
		if k[0] == 'v' {
			sv := k[1:]
			cv, err := strconv.ParseUint(sv, 10, 32)
			if err != nil || cv > 0x7FffFFff {
				return fmt.Errorf("invalid v version %q", sv)
			}
			tool.migrations = append(tool.migrations, schemaAndVer{
				s: v,
				v: int(cv),
			})
			continue
		}
		if k[0] == 's' {
			sv := k[1:]
			cv, err := strconv.ParseUint(sv, 10, 32)
			if err != nil || cv > 0x7FffFFff {
				return fmt.Errorf("invalid s version %q", sv)
			}
			tool.seeds = append(tool.seeds, schemaAndVer{
				s: v,
				v: int(cv),
			})
			continue
		}
		return fmt.Errorf("unknown item %q", k)
	}

	sort.Slice(tool.migrations, func(i, j int) bool {
		return tool.migrations[i].v < tool.migrations[j].v
	})
	sort.Slice(tool.seeds, func(i, j int) bool {
		return tool.seeds[i].v < tool.seeds[j].v
	})
	for i := 1; i < len(tool.seeds); i++ {
		if tool.seeds[i-1].v == tool.seeds[i].v {
			return fmt.Errorf(
				"invalid config: duplicate seed entry")
		}
	}
	for i := 1; i < len(tool.migrations); i++ {
		if tool.migrations[i-1].v == tool.migrations[i].v {
			return fmt.Errorf(
				"invalid config: duplicate migration entry")
		}
	}

	if tool.current != nil {
		ms := tool.seeds[len(tool.seeds)-1]
		if ms.v > currentVer {
			return fmt.Errorf(
				"invalid config: current ver %d < seed ver %d", currentVer, ms.v)
		}
		if ms.v == currentVer && !reflect.DeepEqual(ms.s, tool.current) {
			return fmt.Errorf(
				"invalid config: current and seed mismatch")
		}
		if mmv := migrations[len(migrations)-1].v; mmv > currentVer {
			return fmt.Errorf(
				"invalid config: current ver %d < migration ver %d", currentVer, mmv)
		}
	}

	tool.maxVer = currentVer
	if msv := seeds[len(seeds)-1].v; msv > tool.maxVer {
		tool.maxVer = msv
	}
	if mmv := migrations[len(migrations)-1].v; mmv > tool.maxVer {
		tool.maxVer = mmv
	}

	return tool, nil
}

var ErrNeedsUpgrade = errors.New("database needs update")

func (tool *PGXSchemaTool) CheckDBVersionConn(conn *pgx.Conn, comp string) error {
	nowVer, err := getVersion(conn, `SELECT version FROM public.capabilities WHERE component = $1`, comp)
	if err != nil {
		return err
	}
	if nowVer > tool.maxVer {
		return fmt.Errorf("database version higher than our (db: %d, our: %d)", nowVer, tool.maxVer)
	}
	if nowVer == tool.maxVer {
		return nil
	}
	for _, m := range tool.migrations {
		if nowVer+1 < m.v {
			// migration version too low
			continue
		}
		// migration version seems either equal or higher
		nowVer++
		if nowVer > m.v {
			return fmt.Errorf("database needs update to %d version, but we can't perform it", nowVer)
		}
		// nowVer == m.v
	}
	if nowVer != tool.maxVer {
		return fmt.Errorf("database needs update to %d version, but we can't perform it", nowVer)
	}
	// we could do it
	return ErrNeedsUpgrade
}

var errVersionRace = errors.New("version race")

func (tool *PGXSchemaTool) UpgradeDBVersionConn(conn *pgx.Conn, comp string) (didSomething bool, err error) {

reVer:

	nowVer, err := getVersion(conn, `SELECT version FROM public.capabilities WHERE component = $1`, comp)
	if err != nil {
		return
	}
	c := 1
	if nowVer < 0 {
		if tool.current != nil {
			c = tool.maxVer-nowVer
		}
	}
	// first check whether we actually can perform upgrade
	if nowVer > tool.maxVer {
		err = fmt.Errorf("database version higher than our (db: %d, our: %d)", nowVer, tool.maxVer)
	}
	if nowVer == tool.maxVer {
		return nil
	}
	for _, m := range tool.migrations {
		if nowVer+c < m.v {
			// migration version too low
			continue
		}
		// migration version seems either equal or higher
		if nowVer+c > m.v {
			err = fmt.Errorf("database needs update to %d version, but we can't perform it", nowVer+c)
		}
		// nowVer+c == m.v
		c++
	}
	if nowVer+c != tool.maxVer {
		err = fmt.Errorf("database needs update to %d version, but we can't perform it", nowVer)
		return
	}

	err = tool.performUpgrade(conn, comp, nowVer)
	if err != nil {
		if err == errVersionRace {
			goto reVer
		}
		return
	}

	didSomething = true
	return
}

func (tool *PGXSchemaTool) performUpgrade(conn *pgx.Conn, comp string, nowVer int) (err error) {

	tx, err := conn.BeginTx(context.Background(), pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return err
	}
	defer func(){
		if err != nil {
			tx.Rollback(context.Background())
		}
	}()

	nowVer2, err := getVersion(tx, `SELECT version FROM public.capabilities WHERE component = $1 FOR UPDATE`, comp)
	if err != nil {
		return err
	}
	if nowVer != nowVer2 {
		err = errVersionRace
		return
	}

	_, err = tx.Exec(context.Background(), `UPDATE public.capabilities SET version=$2 WHERE component = $1`, comp, tool.maxVer)
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

func executeMigration(tx pgx.Tx, s []string) error {
	for _, v := range s {
		_, err := tx.Exec(context.Background(), v, pgx.QuerySimpleProtocol(true))
		if err != nil {
			return err
		}
	}
	return nil
}

type pgxQueryer interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func getVersion(q pgxQueryer, stmt, comp string) (_ int, err error) {
	var ver int
	stmt :=
	err = q.QueryRow(context.Background(), stmt, pgx.QuerySimpleProtocol(true), comp).Scan(&ver)
	if err != nil {
		if err == pgx.ErrNoRows {
			return -1, nil
		}
		return -1, err
	}
	return ver, nil
}

func CheckServerVersion(q pgxQueryer, verReq int) error {
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
