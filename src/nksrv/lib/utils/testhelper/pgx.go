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

	dPath string
	contID string
}

func newDockerPGXProvider(dPath, port string) (_ PGXProvider, err error) {

	pgImage := "postgres:alpine"

	err = exec.Command(dPath, "pull", pgImage).Run()
	if err != nil {
		return
	}

	var buf [12]byte
	_, err = rand.Read(buf[:])
	if err != nil {
		panic("rand.Read err: " + err.Error())
	}
	passwd := fmt.Sprintf("%x", buf)

	bid, err := exec.Command(dPath, "run", "-e", "POSTGRES_PASSWORD="+passwd, "-d", pgImage).Output()
	if err != nil {
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

	defer func(){
		if err != nil {
			_ = exec.Command(dPath, "rm", "-v", id).Run()
		}
	}()

	conn, err := connPGXString("host=localhost dbname=postgres user=postgres password=" + passwd)
	if err != nil {
		return
	}

	return dockerPGXProvider{
		directPGXProvider: directPGXProvider{
			conn: conn,
		},
		dPath: dPath,
		contID: id,
	}, nil
}

func (pp dockerPGXProvider) Close() error {
	err := pp.directPGXProvider.Close()
	e := exec.Command(pp.dPath, "rm", "-v", pp.contID).Run()
	if e != nil && err == nil {
		err =  e
	}
	return err
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


type PGXDatabase struct {
	mainConn *pgx.Conn
	Config *pgx.ConnConfig
}

func (pp directPGXProvider) Close() error {
	return pp.conn.Close(context.Background())
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
	defer func(){
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


func createPGXDatabase(conn *pgx.Conn, name string) error {
	_, err := conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", name))
	return err
}

func dropPGXDatabase(conn *pgx.Conn, name string) error {
	_, err := conn.Exec(context.Background(), fmt.Sprintf("DROP DATABASE %s", name))
	return err
}


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
		conn, err := pgx.ConnectConfig(context.Background(), cfg)
		if err != nil {
			if c >= 15 {
				return err
			}
			c++
			time.Sleep(time.Millisecond * 200)
			continue
		}

		var dummy int
		err = conn.QueryRow(context.Background(), "SELECT 1").Scan(&dummy)
		_ = conn.Close(context.Background())
		if err != nil {
			if c >= 15 {
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
		conn, err = pgx.ConnectConfig(context.Background(), cfg)
		if err != nil {
			if c >= 15 {
				return
			}
			c++
			time.Sleep(time.Millisecond * 200)
			continue
		}

		var dummy int
		err = conn.QueryRow(context.Background(), "SELECT 1").Scan(&dummy)
		if err != nil {
			_ = conn.Close(context.Background())

			if c >= 15 {
				return
			}
			c++
			time.Sleep(time.Millisecond * 200)
			continue
		}

		return conn, nil
	}
}
