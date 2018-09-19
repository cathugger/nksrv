package psqlib

// database stuff

import (
	"database/sql"
	"fmt"
)

var currIb0Version = ""

var dbIb0InitStatements = []string{
	`CREATE SCHEMA IF NOT EXISTS ib0`,
	`CREATE TABLE ib0.boards (
	bname  TEXT    NOT NULL,           -- external board identifier
	bid    SERIAL  NOT NULL,           -- internal board ID
	lastid BIGINT  DEFAULT 0 NOT NULL, -- used for post/thread IDs

	dadded TIMESTAMP WITHOUT TIME ZONE NOT NULL, -- date added to our node

	threads_per_page INTEGER, -- <=0 - infinite, this results in only single page
	max_active_pages INTEGER, -- <=0 - all existing pages are active
	max_pages        INTEGER, -- <=0 - unlimited, archive mode

	post_limits      JSONB, /* allowed properties of post, sorta common for both OPs and replies */
	newthread_limits JSONB, /* same as post_limits but for new threads. inherits from post_limits */
	reply_limits     JSONB, /* same as post_limits but for replies. inherits from post_limits */
	thread_opts      JSONB, /* options common for all threads. stuff like bump/file limits */
	attrib           JSONB, /* board attributes */

	UNIQUE      (bname),
	PRIMARY KEY (bid)
)`,
	`CREATE TABLE ib0.threads (
	bid   INTEGER                     NOT NULL, /* internal board ID this thread belongs to */
	tname TEXT                        NOT NULL, /* external thread identifier */
	tid   BIGINT                      NOT NULL, /* internal thread ID */
	bump  TIMESTAMP WITHOUT TIME ZONE NOT NULL, /* last bump time. decides position in pages/catalog */

	reply_limits JSONB, /* inherits from reply_limits of ib0.boards */
	thread_opts  JSONB, /* inherits from thread_opts of ib0.boards */
	attrib       JSONB, /* extra attributes */

	UNIQUE      (bid,tname),
	PRIMARY KEY (bid,tid),
	FOREIGN KEY (bid) REFERENCES ib0.boards
)`,
	`CREATE TABLE ib0.posts (
	bid     INTEGER                     NOT NULL, /* internal board ID this post belongs to */
	pname   TEXT                        NOT NULL, /* extermal post identifier */
	pid     BIGINT                      NOT NULL, /* internal post ID of this post. if pid==tid then this is OP */
	tid     BIGINT                      NOT NULL, /* internal thread ID this post belongs to */
	padded  TIMESTAMP WITHOUT TIME ZONE NOT NULL, /* date field used for sorting. will actually contain delivery date */
	pdate   TIMESTAMP WITHOUT TIME ZONE NOT NULL, /* real date field */
	sage    BOOLEAN                     NOT NULL, /* if true this isn't bump */

	msgid   TEXT                        NOT NULL, /* Message-ID */
	author  TEXT                        NOT NULL, /* author name */
	trip    TEXT                        NOT NULL, /* XXX should we have it there and not in attrib? probably yes, we could benefit from search */
	title   TEXT                        NOT NULL, /* message title/subject field */
	message TEXT,                                 /* post message, in UTF-8 */
	attrib  JSONB,                                /* extra attributes which are optional */
	extras  JSONB,                                /* dunno if really need this field */

	UNIQUE      (msgid),
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
	ftype    TEXT      NOT NULL, /* file type */
	fsize    BIGINT    NOT NULL, /* file size */
	fname    TEXT      NOT NULL, /* filename of original file. not unique! */
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

func (sp *PSQLIB) InitIb0() {
	for i := range dbIb0InitStatements {
		sp.db.DB.MustExec(dbIb0InitStatements[i])
	}
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
