-- :name version
demo2


-- :name init
INSERT INTO capabilities(component,version) VALUES ('ib0','demo3')


-- :next
CREATE SCHEMA ib0


-- :next
CREATE TABLE ib0.boards (
	b_id    SERIAL             NOT NULL, -- internal board ID
	b_name  TEXT               NOT NULL, -- external board identifier
	last_id BIGINT  DEFAULT 0  NOT NULL, -- used for post/thread IDs

	t_count BIGINT  DEFAULT 0  NOT NULL, -- thread count
	p_count BIGINT  DEFAULT 0  NOT NULL, -- post count

	badded TIMESTAMP  WITHOUT TIME ZONE  NOT NULL, -- date added to our node
	bdesc  TEXT                          NOT NULL, -- short description

	threads_per_page INTEGER, -- <=0 - infinite, this results in only single page
	max_active_pages INTEGER, -- <=0 - all existing pages are active
	max_pages        INTEGER, -- <=0 - unlimited, archive mode

	post_limits      JSONB, -- allowed properties of post, sorta common for both OPs and replies
	newthread_limits JSONB, -- same as post_limits but for new threads. inherits from post_limits
	reply_limits     JSONB, -- same as post_limits but for replies. inherits from post_limits
	thread_opts      JSONB, -- options common for all threads. stuff like bump/file limits
	attrib           JSONB, -- board attributes

	PRIMARY KEY (b_id),
	UNIQUE      (b_name)
)
-- :next
CREATE INDEX
	ON ib0.boards (badded,b_id) -- NEWGROUPS


-- :next
CREATE TABLE ib0.threads (
	b_id   INTEGER                     NOT NULL, -- internal board ID this thread belongs to
	t_id   BIGINT                      NOT NULL, -- internal thread ID
	t_name TEXT     COLLATE ucs_basic  NOT NULL, -- external thread identifier

	bump    TIMESTAMP  WITHOUT TIME ZONE  NOT NULL, -- last bump time. decides position in pages/catalog
	p_count BIGINT                        NOT NULL, -- post count
	f_count BIGINT                        NOT NULL, -- sum of posts' f_count

	reply_limits JSONB, -- inherits from reply_limits of ib0.boards
	thread_opts  JSONB, -- inherits from thread_opts of ib0.boards
	attrib       JSONB, -- extra attributes

	PRIMARY KEY (b_id,t_id),
	UNIQUE      (b_id,t_name),
	FOREIGN KEY (b_id) REFERENCES ib0.boards
)
-- :next
CREATE INDEX
	ON ib0.threads (b_id,bump DESC,t_id ASC)


-- :next
CREATE TABLE ib0.posts (
	g_p_id BIGSERIAL                     NOT NULL, -- global internal post ID
	msgid  TEXT       COLLATE ucs_basic  NOT NULL, -- Message-ID

	-- redundant
	pdate  TIMESTAMP  WITHOUT TIME ZONE  NOT NULL, -- real date field
	padded TIMESTAMP  WITHOUT TIME ZONE  NOT NULL, -- date field used for sorting. will actually contain delivery date
	sage   BOOLEAN                       NOT NULL, -- if true this isn't bump

	f_count INTEGER NOT NULL, -- attachment count

	author  TEXT                         NOT NULL, -- author name
	trip    TEXT      COLLATE ucs_basic  NOT NULL, -- XXX should we have it there and not in attrib? probably yes, we could benefit from search
	title   TEXT                         NOT NULL, -- message title/subject field
	message TEXT                         NOT NULL, -- post message, in UTF-8
	headers JSONB,                                 -- map of lists of strings
	attrib  JSONB,                                 -- extra attributes which are optional
	layout  JSONB,                                 -- multipart msg and attachment layout
	extras  JSONB,                                 -- dunno if really need this field

	PRIMARY KEY (g_p_id),
	UNIQUE      (msgid)
)


-- :next
CREATE TABLE ib0.bposts (
	b_id   INTEGER                     NOT NULL, -- internal board ID this post belongs to
	b_p_id BIGINT                      NOT NULL, -- internal post ID of this post. if pid==tid then this is OP
	p_name TEXT     COLLATE ucs_basic  NOT NULL, -- external post identifier
	t_id   BIGINT                      NOT NULL, -- internal thread ID this post belongs to
	g_p_id BIGINT                      NOT NULL, -- global internal post ID

	-- redundant
	pdate  TIMESTAMP  WITHOUT TIME ZONE  NOT NULL, -- real date field
	padded TIMESTAMP  WITHOUT TIME ZONE  NOT NULL, -- date field used for sorting. will actually contain delivery date
	sage   BOOLEAN                       NOT NULL, -- if true this isn't bump

	PRIMARY KEY (b_id,b_p_id),
	UNIQUE      (g_p_id,b_id),
	FOREIGN KEY (b_id)      REFERENCES ib0.boards,
	FOREIGN KEY (b_id,t_id) REFERENCES ib0.threads,
	FOREIGN KEY (g_p_id)    REFERENCES ib0.posts
)
-- :next
CREATE INDEX
	ON ib0.bposts (
		b_id,
		t_id,
		pdate ASC,
		b_p_id ASC
	) -- in thread, for bump
-- :next
CREATE INDEX
	ON ib0.bposts (padded,g_p_id,b_id) -- for NEWNEWS
-- :next
CREATE UNIQUE INDEX
	ON ib0.bposts (p_name text_pattern_ops,b_id) -- for post num lookup


-- :next
CREATE TYPE ftype_t AS ENUM ('file', 'msg', 'text', 'image')
-- :next
CREATE TABLE ib0.files (
	f_id   BIGSERIAL NOT NULL, -- internal file ID of this file
	g_p_id BIGINT    NOT NULL, -- post file belongs to

	fname    TEXT     COLLATE ucs_basic  NOT NULL, -- internal file name of original file. not unique!
	ftype    ftype_t                     NOT NULL, -- file type
	fsize    BIGINT                      NOT NULL, -- file size
	thumb    TEXT     COLLATE ucs_basic  NOT NULL, -- filename of thumbnail. not unique!
	oname    TEXT     COLLATE ucs_basic  NOT NULL, -- original file name of this file
	filecfg  JSONB,                                -- additional info about original file. like metadata
	thumbcfg JSONB,                                -- additional info about thumbnail. like width/height
	extras   JSONB,                                -- extra info not used for display but sometimes useful. undecided.

	PRIMARY KEY (f_id),
	FOREIGN KEY (g_p_id) REFERENCES ib0.posts
)
-- :next
CREATE INDEX ON ib0.files (g_p_id)
-- :next
CREATE INDEX ON ib0.files (fname)


-- :next
-- index of failed references, so that we can pick them up and correct
CREATE TABLE ib0.failrefs (
	fr_id BIGSERIAL NOT NULL,

	g_p_id BIGINT NOT NULL,

	p_name TEXT  COLLATE ucs_basic, -- external post identifier
	b_name TEXT                   ,
	msgid  TEXT  COLLATE ucs_basic, -- Message-ID

	FOREIGN KEY (g_p_id)
		REFERENCES ib0.posts
		ON DELETE CASCADE
)
-- :next
CREATE INDEX
	ON ib0.failrefs (g_p_id)
-- :next
CREATE INDEX
	ON ib0.failrefs(p_name text_pattern_ops,b_name NULLS FIRST)
	WHERE p_name IS NOT NULL
-- :next
CREATE INDEX
	ON ib0.failrefs(msgid)
	WHERE msgid IS NOT NULL



-- :next
CREATE TABLE ib0.scraper_list (
	sid      BIGSERIAL                     NOT NULL,
	sname    TEXT       COLLATE ucs_basic  NOT NULL,
	last_use BIGINT                        NOT NULL, -- used for cleanup

	PRIMARY KEY (sid),
	UNIQUE (sname)
)
-- :next
CREATE INDEX ON ib0.scraper_list (last_use)


-- :next
CREATE TABLE ib0.scraper_last_newnews (
	sid          BIGINT NOT NULL,
	last_newnews BIGINT NOT NULL,

	PRIMARY KEY (sid),
	FOREIGN KEY (sid)
		REFERENCES ib0.scraper_list
		ON DELETE CASCADE
)


-- :next
CREATE TABLE ib0.scraper_last_newgroups (
	sid            BIGINT NOT NULL,
	last_newgroups BIGINT NOT NULL,

	PRIMARY KEY (sid),
	FOREIGN KEY (sid)
		REFERENCES ib0.scraper_list
		ON DELETE CASCADE
)


-- :next
CREATE TABLE ib0.scraper_group_track (
	sid      BIGINT  NOT NULL,
	bid      INTEGER NOT NULL,
	last_use BIGINT  NOT NULL, -- used for cleanup
	last_max BIGINT  NOT NULL, -- max id seen last time
	next_max BIGINT  NOT NULL, -- new max id

	PRIMARY KEY (sid,bid),
	FOREIGN KEY (sid)
		REFERENCES ib0.scraper_list
		ON DELETE CASCADE,
	FOREIGN KEY (bid)
		REFERENCES ib0.boards
		ON DELETE CASCADE
)
-- :next
CREATE INDEX ON ib0.scraper_group_track (sid,last_use)
