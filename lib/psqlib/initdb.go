package psqlib

import (
	"database/sql"
	"fmt"
)

var (
	currDBVersion   = ""
	currCom0Version = ""
	currIb0Version  = ""
)

var dbInitStatements = []string{
	`CREATE TABLE capabilities (
	component TEXT NOT NULL UNIQUE,
	version   TEXT NOT NULL
)`,
	`INSERT INTO capabilities(component,version) VALUES ('','')`,
}

var dbCom0InitStatements = []string{
	`CREATE SCHEMA IF NOT EXISTS com0`,
	`CREATE TABLE com0.groups (
	gname     TEXT  NOT NULL, /* group name */
	bend_type TEXT  NOT NULL, /* backend type */
	bend_cfg  JSONB NOT NULL, /* backend config. format depends on backend type */
	PRIMARY KEY (gname)
)`,
	`INSERT INTO capabilities(component,version) VALUES ('com0','')`,
}

var dbIb0InitStatements = []string{
	`CREATE SCHEMA IF NOT EXISTS ib0`,
	`CREATE TABLE ib0.boards (
	bname  TEXT   NOT NULL,           /* external board identifier */
	bid    SERIAL NOT NULL,           /* internal board ID */
	attrib JSONB,                     /* board attributes */
	lastid BIGINT DEFAULT 0 NOT NULL, /* used for post/thread IDs */
	UNIQUE      (bname),
	PRIMARY KEY (bid)
)`,
	`CREATE TABLE ib0.threads (
	bid    INTEGER                     NOT NULL, /* internal board ID this thread belongs to */
	tname  TEXT                        NOT NULL, /* external thread identifier */
	tid    BIGINT                      NOT NULL, /* internal thread ID */
	bump   TIMESTAMP WITHOUT TIME ZONE NOT NULL, /* last bump time. decides position in pages/catalog */
	attrib JSONB,                                /* extra attributes */
	UNIQUE      (bid,tname),
	PRIMARY KEY (bid,tid),
	FOREIGN KEY (bid) REFERENCES ib0.boards
)`,
	`CREATE TABLE ib0.posts (
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
)`,
	`CREATE INDEX ON ib0.posts (bid)`,
	`CREATE INDEX ON ib0.posts (bid,tid)`,
	`CREATE TABLE ib0.files (
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
)`,
	`CREATE INDEX ON ib0.files (bid,pid)`,
	`CREATE INDEX ON ib0.files (fname)`,
	`INSERT INTO capabilities(component,version) VALUES ('ib0','')`,
}

func (sp *PSQLIB) InitDB() {
	for i := range dbInitStatements {
		sp.db.DB.MustExec(dbInitStatements[i])
	}
}

func (sp *PSQLIB) InitCom0() {
	for i := range dbCom0InitStatements {
		sp.db.DB.MustExec(dbCom0InitStatements[i])
	}
}

func (sp *PSQLIB) InitIb0() {
	for i := range dbIb0InitStatements {
		sp.db.DB.MustExec(dbIb0InitStatements[i])
	}
}

func (sp *PSQLIB) IsValidDB() bool {
	var exists bool
	err := sp.db.DB.
		QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'capabilities')`).
		Scan(&exists)
	if err != nil {
		panic(fmt.Sprintf("error checking validity: %v", err))
	}
	return exists
}

func (sp *PSQLIB) CheckVersion() error {
	var ver string
	err := sp.db.DB.
		QueryRow("SELECT version FROM capabilities WHERE component = '' LIMIT 1").
		Scan(&ver)
	if err != nil {
		return sp.sqlError("version row query", err)
	}
	if ver != currDBVersion {
		return fmt.Errorf("incorrect database version: %q (our: %q)", ver, currDBVersion)
	}
	return nil
}

func (sp *PSQLIB) CheckCom0() (initialised bool, versionerror error) {
	var ver string
	err := sp.db.DB.
		QueryRow("SELECT version FROM capabilities WHERE component = 'ib0' LIMIT 1").
		Scan(&ver)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sp.sqlError("version row query", err)
	}
	if ver != currCom0Version {
		return true, fmt.Errorf("incorrect ib0 schema version: %q (our: %q)", ver, currCom0Version)
	}
	return true, nil
}

func (sp *PSQLIB) CheckIb0() (initialised bool, versionerror error) {
	var ver string
	err := sp.db.DB.
		QueryRow("SELECT version FROM capabilities WHERE component = 'ib0' LIMIT 1").
		Scan(&ver)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sp.sqlError("version row query", err)
	}
	if ver != currIb0Version {
		return true, fmt.Errorf("incorrect ib0 schema version: %q (our: %q)", ver, currIb0Version)
	}
	return true, nil
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
