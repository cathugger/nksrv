package psqlib

import "fmt"

var currPSQLVersion uint32 = 0

var initStatement0 = `CREATE SCHEMA com0`
var initStatement1 = `CREATE TABLE com0.version (ver INTEGER)`
var initStatement2 = `INSERT INTO com0.version(ver) VALUES (0)`
var initStatement3 = `CREATE TABLE com0.groups (
	gname     TEXT  NOT NULL, /* group name */
	bend_type TEXT  NOT NULL, /* backend type */
	bend_cfg  JSONB NOT NULL, /* backend config. format depends on backend type */
	PRIMARY KEY (gname)
)`
var initStatement4 = `CREATE SCHEMA ib0`
var initStatement5 = `CREATE TABLE ib0.boards (
	bname  TEXT   NOT NULL,           /* external board identifier */
	bid    SERIAL NOT NULL,           /* internal board ID */
	attrib JSONB,                     /* board attributes */
	lastid BIGINT DEFAULT 0 NOT NULL, /* used for post/thread IDs */
	UNIQUE      (bname),
	PRIMARY KEY (bid)
)`
var initStatement6 = `CREATE TABLE ib0.threads (
	bid    INTEGER                     NOT NULL, /* internal board ID this thread belongs to */
	tname  TEXT                        NOT NULL, /* external thread identifier */
	tid    BIGINT                      NOT NULL, /* internal thread ID */
	bump   TIMESTAMP WITHOUT TIME ZONE NOT NULL, /* last bump time. decides position in pages/catalog */
	attrib JSONB,                                /* extra attributes */
	UNIQUE      (bid,tname),
	PRIMARY KEY (bid,tid),
	FOREIGN KEY (bid) REFERENCES ib0.boards
)`
var initStatement7 = `CREATE TABLE ib0.posts (
	bid     INTEGER                     NOT NULL, /* internal board ID this post belongs to */
	pname   TEXT                        NOT NULL, /* extermal post identifier */
	pid     BIGINT                      NOT NULL, /* internal post ID of this post. if pid==tid then this is OP */
	tid     BIGINT                      NOT NULL, /* internal thread ID this post belongs to */
	author  TEXT                        NOT NULL, /* author name */
	trip    TEXT                        NOT NULL, /* XXX should we have it there and not in attrib? */
	email   TEXT                        NOT NULL, /* XXX should we even have this? */
	subject TEXT                        NOT NULL, /* message subject field */
	pdate   TIMESTAMP WITHOUT TIME ZONE NOT NULL, /* date field used for sorting. may actually contain delivery (not creation) date */
	message TEXT,                                 /* post message, in UTF-8 */
	attrib  JSONB,                                /* extra attributes */
	extras  JSONB,                                /* stuff not usually queried by frontends but needed to restore original message; also useless meta shit like poster address */
	UNIQUE      (bid,pname),
	PRIMARY KEY (bid,pid),
	FOREIGN KEY (bid)     REFERENCES ib0.boards,
	FOREIGN KEY (bid,tid) REFERENCES ib0.threads
)`
var initStatement8 = `CREATE TABLE ib0.files (
	fid      BIGSERIAL NOT NULL, /* internal file ID of this file */
	bid      INTEGER   NOT NULL, /* internal board ID post of this file belongs to */
	pid      BIGINT    NOT NULL, /* internal post ID of post this file belongs to */
	fname    TEXT      NOT NULL, /* filename of original file. not unique! */
	ftype    TEXT      NOT NULL, /* file type */
	fsize    BIGINT    NOT NULL, /* file size */
	thumb    TEXT      NOT NULL, /* filename of thumbnail. not unique! */
	oname    TEXT      NOT NULL, /* original filename of this file */
	filecfg  JSONB,              /* additional info about original file */
	thumbcfg JSONB,              /* additional info about thumbnail */
	extras   JSONB,              /* extra info not used for display but sometimes useful */
	PRIMARY KEY (fid),
	FOREIGN KEY (bid)     REFERENCES ib0.boards,
	FOREIGN KEY (bid,pid) REFERENCES ib0.posts
)`
var initStatement9 = `CREATE INDEX ON ib0.files (bid,pid)`
var initStatement10 = `CREATE INDEX ON ib0.files (fname)`

func (sp *PSQLIB) initDB() {
	sp.db.DB.MustExec(initStatement0)
	sp.db.DB.MustExec(initStatement1)
	sp.db.DB.MustExec(initStatement2)
	sp.db.DB.MustExec(initStatement3)
	sp.db.DB.MustExec(initStatement4)
	sp.db.DB.MustExec(initStatement5)
	sp.db.DB.MustExec(initStatement6)
	sp.db.DB.MustExec(initStatement7)
	sp.db.DB.MustExec(initStatement8)
	sp.db.DB.MustExec(initStatement9)
	sp.db.DB.MustExec(initStatement10)
}

func (sp *PSQLIB) checkVersion() error {
	var ver uint32
	err := sp.db.DB.
		QueryRow("SELECT ver FROM com0.version LIMIT 1").
		Scan(&ver)
	if err != nil {
		return sp.sqlError("version row query", err)
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
