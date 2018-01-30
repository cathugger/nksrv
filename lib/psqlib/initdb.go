package psqlib

import "fmt"

var currPSQLVersion uint32 = 0

var initStatement0 = `CREATE SCHEMA com0`
var initStatement1 = `CREATE TABLE com0.version (ver INTEGER)`
var initStatement2 = `INSERT INTO com0.version(ver) VALUES (0)`
var initStatement3 = `CREATE TABLE com0.groups (
	gname     TEXT NOT NULL PRIMARY KEY,
	bend_type TEXT NOT NULL,
	bend_cfg  JSONB NOT NULL
)`
var initStatement4 = `CREATE SCHEMA ib0`
var initStatement5 = `CREATE TABLE ib0.boards (
	bname  TEXT NOT NULL PRIMARY KEY,
	bid    SERIAL NOT NULL UNIQUE,
	attrib JSONB,
	lastid BIGINT DEFAULT 0 NOT NULL
)`
var initStatement6 = `CREATE TABLE ib0.threads (
	bid    INTEGER NOT NULL REFERENCES ib0.boards(bid),
	tname  TEXT NOT NULL,
	tid    BIGINT NOT NULL,
	bump   TIMESTAMP WITHOUT TIME ZONE NOT NULL,
	attrib JSONB,
	PRIMARY KEY (bid,tname),
	UNIQUE (bid,tid)
)`
var initStatement7 = `CREATE TABLE ib0.posts (
	bid     INTEGER NOT NULL REFERENCES ib0.boards(bid),
	pname   TEXT NOT NULL,
	pid     BIGINT NOT NULL,
	tid     BIGINT NOT NULL,
	author  TEXT NOT NULL,
	trip    TEXT NOT NULL,
	email   TEXT NOT NULL,
	subject TEXT NOT NULL,
	pdate   TIMESTAMP WITHOUT TIME ZONE NOT NULL,
	message TEXT,
	attrib  JSONB,
	extras  JSONB,
	PRIMARY KEY (bid,pname),
	UNIQUE (bid,pid),
	FOREIGN KEY (bid,tid) REFERENCES ib0.threads(bid,tid)
)`
var initStatement8 = `CREATE TABLE ib0.files (
	fid      BIGSERIAL NOT NULL PRIMARY KEY,
	bid      INTEGER NOT NULL REFERENCES ib0.boards(bid),
	pid      BIGINT NOT NULL,
	fname    TEXT NOT NULL,
	thumb    TEXT NOT NULL,
	oname    TEXT NOT NULL,
	filecfg  JSONB,
	thumbcfg JSONB,
	extras   JSONB,
	FOREIGN KEY (bid,pid) REFERENCES ib0.posts(bid,pid)
)`

func (sp PSQLIB) initDB() {
	sp.db.MustExec(initStatement0)
	sp.db.MustExec(initStatement1)
	sp.db.MustExec(initStatement2)
	sp.db.MustExec(initStatement3)
	sp.db.MustExec(initStatement4)
	sp.db.MustExec(initStatement5)
	sp.db.MustExec(initStatement6)
	sp.db.MustExec(initStatement7)
	sp.db.MustExec(initStatement8)
}

func (sp PSQLIB) checkVersion() error {
	var ver uint32
	err := sp.db.
		QueryRow("SELECT ver FROM com0.version LIMIT 1").
		Scan(&ver)
	if err != nil {
		return sqlError(err, "row query")
	}
	if ver != currPSQLVersion {
		return fmt.Errorf("incorrect database version: %d (our: %d)", ver, currPSQLVersion)
	}
	return nil
}

// new thread without files
var _ = `
WITH
	ub AS (
		UPDATE ib0.boards
		SET lastid = lastid+1
		WHERE bname='testb'
		RETURNING lastid,bid
	),
	ut AS (
		INSERT INTO ib0.threads (bid,tname,tid,bump)
		SELECT bid,'0123456789ABCDEEF',lastid,NOW()
		FROM ub
		RETURNING bid,tid,bump
	),
	up AS (
		INSERT INTO ib0.posts (bid,pname,pid,tid,author,trip,email,subject,pdate,message)
		SELECT bid,'0123456789ABCDEEF',tid,tid,'','','','test subject',bump,'test message'
		FROM ut
		RETURNING pid
	)
SELECT * FROM up;
`

// new thread + bulk file insert
var _ = `
WITH
	ub AS (
		UPDATE ib0.boards
		SET lastid = lastid+1
		WHERE bname='testb'
		RETURNING lastid,bid
	),
	ut AS (
		INSERT INTO ib0.threads (bid,tname,tid,bump)
		SELECT bid,'0123456789ABCDEEF222',lastid,NOW()
		FROM ub
		RETURNING bid,tid,bump
	),
	up AS (
		INSERT INTO ib0.posts (bid,pname,pid,tid,author,trip,email,subject,pdate,message)
		SELECT bid,'0123456789ABCDEEF222',tid,tid,'','','','test subject',bump,'test message'
		FROM ut
		RETURNING bid,pid
	),
	uf AS (
		INSERT INTO ib0.files (bid,pid,fname,thumb,oname)
		SELECT *
		FROM (
			SELECT bid,pid
			FROM up
		) AS q0
		CROSS JOIN (
			VALUES
				('fname1.jpg', 'tname1.jpg', 'oname1.jpg'),
				('fname2.jpg', 'tname2.jpg', 'oname2.jpg')
		) AS q1
	)
SELECT * FROM up
`