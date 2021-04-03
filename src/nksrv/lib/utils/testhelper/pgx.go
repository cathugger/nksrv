package testhelper

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v4"
)

var mu sync.Mutex

type PGXProvider interface {
	NewDatabase() (PGXDatabase, error)
	Close() error
}

type directPGXProvider struct {
	conn *pgx.Conn
}

func NewPGXProvider() (_ PGXProvider, err error) {
	if e := testTCPPort("localhost:5432"); e == nil {
		// something's running on that port so try to connect
		return newDirectPGXProvider("postgres://postgres@localhost:5432/postgres")
	}

	// else attempt to set postgres up with docker
	if dPath, e := exec.LookPath("docker"); e == nil {
		return newDockerPGXProvider(dPath, "")
	}

	err = errors.New("no provider")
	return
}

type dockerPGXProvider struct {
	directPGXProvider

	dPath  string
	contID string
}

func execError(when string, e error) error {
	if ee, _ := e.(*exec.ExitError); ee != nil {
		return fmt.Errorf(
			"error during execution of %s (%w): %s",
			when, e, strings.TrimSpace(string(ee.Stderr)),
		)
	} else {
		return fmt.Errorf("error executing %s: %w",when, e)
	}
}

func newDockerPGXProvider(dPath, port string) (_ PGXProvider, err error) {

	pgImage := "postgres:alpine"

	err = exec.Command(dPath, "pull", pgImage).Run()
	if err != nil {
		err = execError("docker pull", err)
		return
	}

	var buf [12]byte
	_, err = rand.Read(buf[:])
	if err != nil {
		panic("rand.Read err: " + err.Error())
	}
	passwd := fmt.Sprintf("%x", buf)

	bid, err := exec.Command(
		dPath, "run",
		"-p", "127.0.0.1:5432:5432",
		"-e", "POSTGRES_PASSWORD="+passwd,
		"-d",
		pgImage,
	).Output()
	if err != nil {
		err = execError("docker run", err)
		return
	}
	id := strings.TrimSpace(string(bid))
	matched, e := regexp.MatchString("^[[:alnum:]]{12,}$", id)
	if e != nil {
		panic("regexp.MatchString err: " + e.Error())
	}
	if !matched {
		err = fmt.Errorf("weird docker output: %q", id)
		return
	}

	defer func() {
		if err != nil {
			_ = exec.Command(dPath, "stop", id).Run()
			_ = exec.Command(dPath, "rm", "-v", "-f", id).Run()
		}
	}()

	conn, err := connPGXString("host=127.0.0.1 sslmode=disable dbname=postgres user=postgres password=" + passwd)
	if err != nil {
		err = fmt.Errorf("failed establishing main connection to container: %w", err)
		return
	}

	return dockerPGXProvider{
		directPGXProvider: directPGXProvider{
			conn: conn,
		},
		dPath:  dPath,
		contID: id,
	}, nil
}

func (pp dockerPGXProvider) Close() error {
	err0 := pp.directPGXProvider.Close()
	e1 := exec.Command(pp.dPath, "stop", pp.contID).Run()
	e2 := exec.Command(pp.dPath, "rm", "-v", "-f", pp.contID).Run()

	if err0 != nil {
		return err0
	}
	if e1 != nil {
		return fmt.Errorf("container stop err: %w", e1)
	}
	if e2 != nil {
		return fmt.Errorf("container rm err: %w", e2)
	}

	return nil
}

func newDirectPGXProvider(connStr string) (_ PGXProvider, err error) {
	conn, err := connPGXString(connStr)
	if err != nil {
		return
	}

	return directPGXProvider{
		conn: conn,
	}, nil
}

///

type PGXDatabase struct {
	mainConn *pgx.Conn
	Config   *pgx.ConnConfig
}

func (pp directPGXProvider) Close() error {
	e := pp.conn.Close(context.Background())
	if e != nil {
		return fmt.Errorf("conn.Close err: %w", e)
	}
	return nil
}

func (pp directPGXProvider) NewDatabase() (_ PGXDatabase, err error) {

	mu.Lock()
	defer mu.Unlock()

	t := time.Now().UTC()

	var buf [12]byte
	_, err = rand.Read(buf[:])
	if err != nil {
		panic("rand.Read err: " + err.Error())
	}

	dbName := fmt.Sprintf(
		"test_%02d%02d%02d_%02d%02d%02d_%x",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		buf,
	)

	err = createPGXDatabase(pp.conn, dbName)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			dropPGXDatabase(pp.conn, dbName)
		}
	}()

	newCfg := pp.conn.Config().Copy()
	newCfg.Database = dbName

	err = testPGXConfig(newCfg)
	if err != nil {
		return
	}

	return PGXDatabase{
		mainConn: pp.conn,
		Config:   newCfg,
	}, nil
}

func (pd PGXDatabase) Close() error {
	mu.Lock()
	defer mu.Unlock()
	return dropPGXDatabase(pd.mainConn, pd.Config.Database)
}

///

func createPGXDatabase(conn *pgx.Conn, name string) error {
	_, err := conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", name))
	return err
}

func dropPGXDatabase(conn *pgx.Conn, name string) error {
	_, err := conn.Exec(context.Background(), fmt.Sprintf("DROP DATABASE %s", name))
	return err
}

///

const cMax = 30

func testTCPPort(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func testPGXConfig(cfg *pgx.ConnConfig) error {
	c := 0
	for {
		var conn *pgx.Conn
		var err error
		func() {
			ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*3)
			defer ctxCancel()
			conn, err = pgx.ConnectConfig(ctx, cfg)
		}()
		if err != nil {
			if c >= cMax {
				return err
			}
			c++
			time.Sleep(time.Millisecond * 200)
			continue
		}

		var dummy int
		func() {
			ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*3)
			defer ctxCancel()
			err = conn.QueryRow(ctx, "SELECT 1").Scan(&dummy)
		}()
		_ = conn.Close(context.Background())
		if err != nil {
			if c >= cMax {
				return err
			}
			c++
			time.Sleep(time.Millisecond * 200)
			continue
		}

		return nil
	}
}

func connPGXString(str string) (_ *pgx.Conn, err error) {
	cfg, err := pgx.ParseConfig(str)
	if err != nil {
		return
	}
	return connPGXConfig(cfg)
}

func connPGXConfig(cfg *pgx.ConnConfig) (_ *pgx.Conn, err error) {
	c := 0

	for {
		var conn *pgx.Conn
		func() {
			ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*3)
			defer ctxCancel()
			conn, err = pgx.ConnectConfig(ctx, cfg)
		}()
		if err != nil {
			//log.Printf("pgx.ConnectConfig err %d: %v", c, err)
			if c >= cMax {
				return
			}
			c++
			time.Sleep(time.Millisecond * 300)
			continue
		}

		var dummy int
		func() {
			ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*3)
			defer ctxCancel()
			err = conn.QueryRow(ctx, "SELECT 1").Scan(&dummy)
		}()
		if err != nil {
			//log.Printf("conn.QueryRow err %d: %v", c, err)
			_ = conn.Close(context.Background())

			if c >= cMax {
				return
			}
			c++
			time.Sleep(time.Millisecond * 300)
			continue
		}

		return conn, nil
	}
}
