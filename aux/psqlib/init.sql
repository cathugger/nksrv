--- core stuff
-- :name version
demo7
-- :name init
CREATE SCHEMA ib0


--- base stuff
-- :next
CREATE TYPE modpriv_t AS ENUM ('none', 'mod')
-- :next
-- design will (very) probably change in the future
CREATE TABLE ib0.modlist (
	mod_id     BIGINT     GENERATED ALWAYS AS IDENTITY,
	mod_pubkey TEXT       COLLATE "C"  NOT NULL,
	automanage BOOLEAN                 NOT NULL,
	mod_priv   modpriv_t               NOT NULL DEFAULT 'none',
	mod_name   TEXT,

	PRIMARY KEY (mod_id),
	UNIQUE      (mod_pubkey)
)


-- :next
CREATE TABLE ib0.posts (
	g_p_id BIGINT  GENERATED ALWAYS AS IDENTITY, -- global internal post ID
	msgid  TEXT    COLLATE "C"  NOT NULL,        -- Message-ID

	-- redundant
	pdate  TIMESTAMP  WITH TIME ZONE, -- real date field
	-- date field used for sorting. will actually contain delivery date
	-- it's not indexed there because we sometimes want to select with board keying
	padded TIMESTAMP  WITH TIME ZONE,
	sage   BOOLEAN                    NOT NULL, -- if true this isn't bump

	f_count INTEGER NOT NULL, -- attachment count

	author  TEXT                 NOT NULL, -- author name
	trip    TEXT    COLLATE "C"  NOT NULL, -- XXX should we have it there and not in attrib? probably yes, we could benefit from search
	title   TEXT                 NOT NULL, -- message title/subject field
	message TEXT                 NOT NULL, -- post message, in UTF-8
	-- headers of msg root, map of lists of strings, needed for NNTP HDR
	headers JSONB,
	-- attributes associated with global post and visible in webui
	attrib  JSON,
	-- article layout, needed to reconstruct original article
	layout  JSON,
	-- passive extra data
	extras  JSONB,

	PRIMARY KEY (g_p_id),
	UNIQUE      (msgid)
)


-- :next
CREATE TABLE ib0.boards (
	b_id    INTEGER  GENERATED ALWAYS AS IDENTITY, -- internal board ID
	b_name  TEXT     COLLATE "C"  NOT NULL,        -- external board identifier
	last_id BIGINT   DEFAULT 0    NOT NULL,        -- used for post/thread IDs

	t_count BIGINT  DEFAULT 0  NOT NULL, -- thread count
	p_count BIGINT  DEFAULT 0  NOT NULL, -- post count

	badded TIMESTAMP  WITH TIME ZONE  NOT NULL, -- date added to our node
	bdesc  TEXT                       NOT NULL, -- short description

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
CREATE INDEX
	ON ib0.boards (b_name COLLATE "und-x-icu")


-- :next
CREATE TABLE ib0.threads (
	b_id     INTEGER               NOT NULL, -- internal board ID this thread belongs to
	b_t_id   BIGINT                NOT NULL, -- internal board-local thread ID (ID of board-local OP post)
	g_t_id   BIGINT                NOT NULL, -- internal global thread OP post ID
	b_t_name TEXT     COLLATE "C"  NOT NULL, -- external thread identifier

	bump      TIMESTAMP  WITH TIME ZONE  NOT NULL, -- last bump time. decides position in pages/catalog
	skip_over BOOLEAN                    NOT NULL, -- if true, do not include in overboard
	p_count   BIGINT                     NOT NULL, -- post count (including OP)
	f_count   BIGINT                     NOT NULL, -- sum of posts' (including OP) f_count
	--fr_count  BIGINT                     NOT NULL, -- file-replies count (not including OP)

	reply_limits JSONB, -- inherits from reply_limits of ib0.boards
	thread_opts  JSONB, -- inherits from thread_opts of ib0.boards
	attrib       JSONB, -- extra attributes

	PRIMARY KEY (b_id,b_t_id),
	UNIQUE      (b_id,b_t_name),
	FOREIGN KEY (b_id)
		REFERENCES ib0.boards
)
-- :next
-- for board pages and catalog
CREATE INDEX
	ON ib0.threads (
		b_id ASC,
		bump DESC,
		b_t_id ASC
	)
-- :next
-- for overboard
CREATE INDEX
	ON ib0.threads (
		bump DESC,
		g_t_id ASC,
		b_id ASC
	)
	WHERE
		skip_over IS NOT TRUE


-- :next
CREATE TABLE ib0.bposts (
	b_id   INTEGER               NOT NULL, -- internal board ID this post belongs to
	b_p_id BIGINT                NOT NULL, -- internal post ID of this post. if pid==tid then this is OP
	p_name TEXT     COLLATE "C"  NOT NULL, -- external post identifier
	b_t_id BIGINT                NOT NULL, -- internal thread ID this post belongs to
	g_p_id BIGINT                NOT NULL, -- global internal post ID
	msgid  TEXT     COLLATE "C"  NOT NULL, -- global external msgid

	-- redundant w/ global but needed for efficient indexes
	pdate  TIMESTAMP  WITH TIME ZONE  NOT NULL, -- real date field
	padded TIMESTAMP  WITH TIME ZONE  NOT NULL, -- date field used for sorting. will actually contain delivery date
	sage   BOOLEAN                    NOT NULL, -- if true this isn't bump

	mod_id BIGINT, -- ID of moderator identity (if ctl msg)
	-- attributes associated with board post and visible in webui
	-- notably, refs
	attrib JSON,

	PRIMARY KEY (b_id,b_p_id),
	UNIQUE      (g_p_id,b_id),
	FOREIGN KEY (b_id)
		REFERENCES ib0.boards,
	FOREIGN KEY (b_id,b_t_id)
		REFERENCES ib0.threads,
	FOREIGN KEY (g_p_id)
		REFERENCES ib0.posts,
	FOREIGN KEY (mod_id)
		REFERENCES ib0.modlist
		ON DELETE RESTRICT
)
-- :next
-- in thread, for bump
CREATE INDEX
	ON ib0.bposts (
		b_id,
		b_t_id,
		pdate ASC,
		b_p_id ASC
	)
-- :next
-- for NEWNEWS (yeh, not in ib0.posts)
CREATE INDEX
	ON ib0.bposts (padded,g_p_id,b_id)
-- :next
-- for post num lookup
CREATE UNIQUE INDEX
	ON ib0.bposts (p_name text_pattern_ops,b_id)
-- :next
-- mostly for boardban checks for now
CREATE UNIQUE INDEX
	ON ib0.bposts (b_id,msgid)
-- :next
-- FK
CREATE INDEX
	ON ib0.bposts (mod_id,pdate)
	WHERE mod_id IS NOT NULL


-- :next
CREATE TYPE ftype_t AS ENUM (
	'file',
	'msg',
	'face',
	'text',
	'image',
	'audio',
	'video'
)
-- :next
CREATE TABLE ib0.files (
	f_id   BIGINT GENERATED ALWAYS AS IDENTITY, -- internal file ID of this file
	g_p_id BIGINT NOT NULL,                     -- post file belongs to

	fname    TEXT     COLLATE "C"  NOT NULL, -- internal file name of original file. not unique!
	ftype    ftype_t               NOT NULL, -- file type
	fsize    BIGINT                NOT NULL, -- file size
	thumb    TEXT     COLLATE "C"  NOT NULL, -- filename of thumbnail. not unique!
	oname    TEXT     COLLATE "C"  NOT NULL, -- original file name of this file
	filecfg  JSON,                           -- additional info about original file. like metadata
	thumbcfg JSON,                           -- additional info about thumbnail. like width/height
	extras   JSON,                           -- extra info not used for display but sometimes useful. undecided.

	PRIMARY KEY (f_id),
	FOREIGN KEY (g_p_id)
		REFERENCES ib0.posts
)
-- :next
CREATE INDEX ON ib0.files (g_p_id,f_id) -- f_id helps sorted retrieval
-- :next
CREATE INDEX ON ib0.files (fname,thumb)


-- :next
-- index of references, so that we can pick them up and correct when we modify stuff
-- references are rendered per-board, not per-post,
-- as multiboard posts may end up refering to odd things otherwise
CREATE TABLE ib0.refs (
	-- board post who owns reference
	b_id   INTEGER NOT NULL,
	b_p_id BIGINT  NOT NULL,

	p_name TEXT  COLLATE "C", -- external post identifier (or part of it)
	b_name TEXT  COLLATE "C", -- board name
	msgid  TEXT  COLLATE "C", -- Message-ID

	FOREIGN KEY (b_id,b_p_id)
		REFERENCES ib0.bposts
		ON DELETE CASCADE
)
-- :next
CREATE INDEX
	ON ib0.refs (b_id,b_p_id) -- FK
-- :next
CREATE INDEX
	ON ib0.refs (p_name text_pattern_ops,b_name NULLS FIRST)
	WHERE p_name IS NOT NULL
-- :next
CREATE INDEX
	ON ib0.refs (b_name)
	WHERE b_name IS NOT NULL AND p_name IS NULL
-- :next
CREATE INDEX
	ON ib0.refs (msgid)
	WHERE msgid IS NOT NULL


--- puller stuff
-- :next
CREATE TABLE ib0.puller_list (
	sid      BIGINT  GENERATED ALWAYS AS IDENTITY,
	sname    TEXT    COLLATE "C"  NOT NULL,
	-- nonce, used to clean dead server trackings
	last_use BIGINT               NOT NULL,

	PRIMARY KEY (sid),
	UNIQUE (sname)
)
-- :next
CREATE INDEX ON ib0.puller_list (last_use)


-- :next
-- last timestamp when we did NEWNEWS (per-server)
CREATE TABLE ib0.puller_last_newnews (
	sid          BIGINT NOT NULL,
	last_newnews BIGINT NOT NULL,

	PRIMARY KEY (sid),
	FOREIGN KEY (sid)
		REFERENCES ib0.puller_list
		ON DELETE CASCADE
)


-- :next
-- last timestamp when we did NEWGROUPS (per-server)
CREATE TABLE ib0.puller_last_newgroups (
	sid            BIGINT NOT NULL,
	last_newgroups BIGINT NOT NULL,

	PRIMARY KEY (sid),
	FOREIGN KEY (sid)
		REFERENCES ib0.puller_list
		ON DELETE CASCADE
)


-- :next
CREATE TABLE ib0.puller_group_track (
	sid      BIGINT  NOT NULL,
	bid      INTEGER NOT NULL,
	-- nonce, used to clean dead board trackings
	last_use BIGINT  NOT NULL,
	-- max id seen last time
	last_max BIGINT  NOT NULL,
	-- max id seen now
	next_max BIGINT  NOT NULL,

	PRIMARY KEY (sid,bid),
	FOREIGN KEY (sid)
		REFERENCES ib0.puller_list
		ON DELETE CASCADE,
	FOREIGN KEY (bid)
		REFERENCES ib0.boards
		ON DELETE CASCADE
)
-- :next
CREATE INDEX
	ON ib0.puller_group_track (sid,last_use)
-- :next
CREATE INDEX
	ON ib0.puller_group_track (bid)


--- moderation stuff
-- :next
CREATE TABLE ib0.banlist (
	ban_id   BIGINT GENERATED ALWAYS AS IDENTITY,
	ban_info TEXT   NOT NULL,

	-- board post responsible for this ban (if any)
	b_id     INTEGER,
	b_p_id   BIGINT,

	msgid  TEXT  COLLATE "C", -- msgid being banned (if any)

	PRIMARY KEY (ban_id),

	FOREIGN KEY (b_id,b_p_id)
		REFERENCES ib0.bposts
		MATCH FULL
		ON DELETE CASCADE    -- see trigger below
)
-- :next
CREATE INDEX
	ON ib0.banlist (b_id,b_p_id)
	WHERE b_id IS NOT NULL AND b_p_id IS NOT NULL
-- :next
CREATE INDEX
	ON ib0.banlist (msgid)
	WHERE msgid IS NOT NULL
-- :next
-- to be ran AFTER delet from banlist
CREATE FUNCTION
	ib0_gc_banposts_after()
RETURNS
	TRIGGER
AS
$$
BEGIN
	-- garbage collect void placeholder posts when all bans for them are lifted
	DELETE FROM
		ib0.posts xp
	USING
		(
			SELECT
				delbl.msgid,COUNT(exibl.msgid) > 0 AS hasrefs
			FROM
				oldrows AS delbl
			LEFT JOIN
				ib0.banlist exibl
			ON
				delbl.msgid = exibl.msgid
			WHERE
				delbl.msgid IS NOT NULL
			GROUP BY
				delbl.msgid
		) AS delp
	WHERE
		delp.hasrefs = FALSE AND delp.msgid = xp.msgid AND xp.padded IS NULL;

	RETURN NULL;
END;
$$
LANGUAGE
	plpgsql
-- :next
CREATE TRIGGER
	ib0_banlist_after_del
AFTER
	DELETE
ON
	ib0.banlist
REFERENCING OLD TABLE AS oldrows
FOR EACH STATEMENT
EXECUTE PROCEDURE
	ib0_gc_banposts_after()
