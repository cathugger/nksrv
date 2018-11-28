package psqlib

// database stuff

import (
	"database/sql"
	"fmt"
)

const currIb0Version = "demo0"

var dbIb0InitStatements = []string{
	`CREATE SCHEMA IF NOT EXISTS ib0`,

	`CREATE TABLE ib0.boards (
	bname  TEXT    NOT NULL,           -- external board identifier
	bid    SERIAL  NOT NULL,           -- internal board ID
	lastid BIGINT  DEFAULT 0 NOT NULL, -- used for post/thread IDs

	badded TIMESTAMP WITHOUT TIME ZONE NOT NULL, -- date added to our node
	bdesc  TEXT NOT NULL, -- short description

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
	`CREATE INDEX ON ib0.boards (badded)`,

	`CREATE TABLE ib0.threads (
	bid   INTEGER                        NOT NULL, /* internal board ID this thread belongs to */
	tname TEXT        COLLATE ucs_basic  NOT NULL, /* external thread identifier */
	tid   BIGINT                         NOT NULL, /* internal thread ID */
	bump  TIMESTAMP WITHOUT TIME ZONE    NOT NULL, /* last bump time. decides position in pages/catalog */

	reply_limits JSONB, /* inherits from reply_limits of ib0.boards */
	thread_opts  JSONB, /* inherits from thread_opts of ib0.boards */
	attrib       JSONB, /* extra attributes */

	UNIQUE      (bid,tname),
	PRIMARY KEY (bid,tid),
	FOREIGN KEY (bid) REFERENCES ib0.boards
)`,

	`CREATE TABLE ib0.posts (
	bid     INTEGER                      NOT NULL, /* internal board ID this post belongs to */
	pname   TEXT      COLLATE ucs_basic  NOT NULL, /* extermal post identifier */
	pid     BIGINT                       NOT NULL, /* internal post ID of this post. if pid==tid then this is OP */
	tid     BIGINT                       NOT NULL, /* internal thread ID this post belongs to */
	padded  TIMESTAMP WITHOUT TIME ZONE  NOT NULL, /* date field used for sorting. will actually contain delivery date */
	pdate   TIMESTAMP WITHOUT TIME ZONE  NOT NULL, /* real date field */
	sage    BOOLEAN                      NOT NULL, /* if true this isn't bump */

	msgid   TEXT      COLLATE ucs_basic  NOT NULL, /* Message-ID */
	author  TEXT                         NOT NULL, /* author name */
	trip    TEXT      COLLATE ucs_basic  NOT NULL, /* XXX should we have it there and not in attrib? probably yes, we could benefit from search */
	title   TEXT                         NOT NULL, /* message title/subject field */
	message TEXT,                                 /* post message, in UTF-8 */
	headers JSONB,                                -- map of lists of strings
	attrib  JSONB,                                /* extra attributes which are optional */
	layout  JSONB,                                /* multipart msg and attachment layout */
	extras  JSONB,                                /* dunno if really need this field */

	UNIQUE      (msgid),
	PRIMARY KEY (bid,pid),
	FOREIGN KEY (bid)     REFERENCES ib0.boards,
	FOREIGN KEY (bid,tid) REFERENCES ib0.threads
)`,
	`CREATE INDEX ON ib0.posts (bid)`,
	`CREATE INDEX ON ib0.posts (bid,tid)`,
	`CREATE INDEX ON ib0.posts (padded,bid)`,
	`CREATE UNIQUE INDEX ON ib0.posts (pname text_pattern_ops,bid)`,

	`CREATE TYPE ftype_t AS ENUM ('file', 'msg', 'text', 'image')`,
	`CREATE TABLE ib0.files (
	fid      BIGSERIAL                   NOT NULL, /* internal file ID of this file */
	bid      INTEGER                     NOT NULL, /* internal board ID post of this file belongs to */
	pid      BIGINT                      NOT NULL, /* internal post ID of post this file belongs to */
	ftype    ftype_t                     NOT NULL, /* file type */
	fsize    BIGINT                      NOT NULL, /* file size */
	fname    TEXT     COLLATE ucs_basic  NOT NULL, /* internal file name of original file. not unique! */
	thumb    TEXT     COLLATE ucs_basic  NOT NULL, /* filename of thumbnail. not unique! */
	oname    TEXT     COLLATE ucs_basic  NOT NULL, /* original file name of this file */
	filecfg  JSONB,                                /* additional info about original file. like metadata */
	thumbcfg JSONB,                                /* additional info about thumbnail. like width/height */
	extras   JSONB,                                /* extra info not used for display but sometimes useful. undecided. */

	PRIMARY KEY (fid),
	FOREIGN KEY (bid)     REFERENCES ib0.boards,
	FOREIGN KEY (bid,pid) REFERENCES ib0.posts
)`,
	`CREATE INDEX ON ib0.files (bid,pid)`,
	`CREATE INDEX ON ib0.files (fname)`,

	`CREATE TABLE ib0.scraper_list (
	sid      BIGSERIAL                     NOT NULL,
	sname    TEXT       COLLATE ucs_basic  NOT NULL,
	last_use BIGINT                        NOT NULL, -- used for cleanup

	PRIMARY KEY (sid),
	UNIQUE (sname)
)`,
	`CREATE INDEX ON ib0.scraper_list (last_use)`,

	`CREATE TABLE ib0.scraper_last_newnews (
	sid          BIGINT NOT NULL,
	last_newnews BIGINT NOT NULL,
	PRIMARY KEY (sid),
	FOREIGN KEY (sid) REFERENCES ib0.scraper_list ON DELETE CASCADE
)`,

	`CREATE TABLE ib0.scraper_last_newgroups (
	sid            BIGINT NOT NULL,
	last_newgroups BIGINT NOT NULL,

	PRIMARY KEY (sid),
	FOREIGN KEY (sid) REFERENCES ib0.scraper_list ON DELETE CASCADE
)`,

	`CREATE TABLE ib0.scraper_group_track (
	sid      BIGINT  NOT NULL,
	bid      INTEGER NOT NULL,
	last_use BIGINT  NOT NULL, -- used for cleanup
	last_max BIGINT  NOT NULL, -- max id seen last time
	next_max BIGINT  NOT NULL, -- new max id

	PRIMARY KEY (sid,bid),
	FOREIGN KEY (sid) REFERENCES ib0.scraper_list ON DELETE CASCADE,
	FOREIGN KEY (bid) REFERENCES ib0.boards ON DELETE CASCADE
)`,
	`CREATE INDEX ON ib0.scraper_group_track (sid,last_use)`,

	`INSERT INTO capabilities(component,version) VALUES ('ib0','` + currIb0Version + `')`,
}

func (sp *PSQLIB) InitIb0() {
	for i := range dbIb0InitStatements {
		_, e := sp.db.DB.Exec(dbIb0InitStatements[i])
		if e != nil {
			panic(fmt.Errorf("err on stmt %d: %v", i, e))
		}
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
