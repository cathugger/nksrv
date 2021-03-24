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

func DoStuffConfig(cfg *pgx.ConnConfig, dir fs.FS) (err error) {
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

	err = DoStuffConn(conn, dir)
	return
}

func DoStuffConn(conn *pgx.Conn, dir fs.FS) error {

	var objects = make(map[string]*[]string)
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
		x := objects[ver]
		if x == nil {
			x = new([]string)
			objects[ver] = x
		}

		*x = append(*x, q["x"][0])

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

	type schemaAndVer struct {
		s *[]string
		v int
	}
	var (
		current    *[]string
		migrations []schemaAndVer
		seeds      []schemaAndVer
	)
	for k, v := range objects {
		if k == "current" {
			current = v
			continue
		}
		if k[0] == 'v' {
			sv := k[1:]
			cv, err := strconv.ParseUint(sv, 10, 32)
			if err != nil || cv > 0x7FffFFff {
				return fmt.Errorf("invalid v version %q", sv)
			}
			migrations = append(migrations, schemaAndVer{
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
			seeds = append(seeds, schemaAndVer{
				s: v,
				v: int(cv),
			})
			continue
		}
		return fmt.Errorf("unknown item %q", k)
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].v < migrations[j].v
	})
	sort.Slice(seeds, func(i, j int) bool {
		return seeds[i].v < seeds[j].v
	})
	for i := 1; i < len(seeds); i++ {
		if seeds[i-1].v == seeds[i].v {
			return fmt.Errorf(
				"invalid config: duplicate seed entry")
		}
	}
	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].v == migrations[i].v {
			return fmt.Errorf(
				"invalid config: duplicate migration entry")
		}
	}
	if current != nil {
		ms := seeds[len(seeds)-1]
		if ms.v > currentVer {
			return fmt.Errorf(
				"invalid config: current ver %d < seed ver %d", currentVer, ms.v)
		}
		if ms.v == currentVer && !reflect.DeepEqual(ms.s, current) {
			return fmt.Errorf(
				"invalid config: current and seed mismatch")
		}
		if mmv := migrations[len(migrations)-1].v; mmv > currentVer {
			return fmt.Errorf(
				"invalid config: current ver %d < migration ver %d", currentVer, mmv)
		}
	}

	maxVer := currentVer
	if seeds[len(seeds)-1].v > maxVer {

	}
}
