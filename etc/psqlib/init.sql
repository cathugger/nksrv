--- core stuff
-- :name version
demo8
-- :name init
CREATE SCHEMA ib0


--- moderators/administrators things
-- :next
-- summary table to hold effective capabilities of moderator
-- mod_cap/mod_bcap/mod_caplvl/mod_bcaplvl changes are logged
CREATE TABLE ib0.modlist (
	mod_id     BIGINT  GENERATED ALWAYS AS IDENTITY,
	mod_pubkey TEXT    COLLATE "C"  NOT NULL,
	mod_name   TEXT,
	-- if true, then no modsets refer to it [so can be GC'd if no posts refer to it too]
	automanage BOOLEAN NOT NULL,
	-- usable capabilities
	mod_cap     BIT(12),        -- global capabilities
	mod_caplvl  SMALLINT ARRAY, -- global cap levels
	mod_bcap    JSONB,          -- per-board capabilities
	mod_bcaplvl JSONB,          -- per-board cap levels
	-- inheritable capabilities
	modi_cap     BIT(12),        -- global capabilities
	modi_caplvl  SMALLINT ARRAY, -- global cap levels
	modi_bcap    JSONB,          -- per-board capabilities
	modi_bcaplvl JSONB,          -- per-board cap levels


	PRIMARY KEY (mod_id),
	UNIQUE      (mod_pubkey)
)


-- :next
CREATE TABLE ib0.gposts (
	g_p_id BIGINT  GENERATED ALWAYS AS IDENTITY, -- global internal post ID
	msgid  TEXT    COLLATE "C"  NOT NULL,        -- Message-ID

	date_sent TIMESTAMP  WITH TIME ZONE,
	date_recv TIMESTAMP  WITH TIME ZONE,
	sage      BOOLEAN    NOT NULL        DEFAULT FALSE,

	-- attachment count
	f_count INTEGER NOT NULL             DEFAULT 0,

	author  TEXT               NOT NULL  DEFAULT '', -- author name
	trip    TEXT  COLLATE "C"  NOT NULL  DEFAULT '', -- XXX should we have it there and not in attrib? probably yes, we could benefit from search
	title   TEXT               NOT NULL  DEFAULT '', -- message title/subject field
	message TEXT               NOT NULL  DEFAULT '', -- post message, in UTF-8

	headers JSONB, -- headers of msg root, map of lists of strings, needed for NNTP HDR
	attrib  JSON,  -- attributes associated with global post and visible in webui
	layout  JSON,  -- article layout, needed to reconstruct original article
	extras  JSONB, -- passive extra data

	mod_dpriv SMALLINT, -- calc'd from bposts

	-- does it have placeholder-related data?
	has_ph BOOLEAN,

	ph_ban     BOOLEAN,
	ph_banpriv SMALLINT,


	PRIMARY KEY (g_p_id),
	UNIQUE      (msgid)
)


-- :next
CREATE TABLE ib0.gposts_boards (
	g_p_id BIGINT               NOT NULL,
	b_name TEXT    COLLATE "C"  NOT NULL,


	FOREIGN KEY (g_p_id)
		REFERENCES ib0.gposts
		ON DELETE CASCADE
		ON UPDATE CASCADE
)
-- :next
CREATE INDEX
	ON ib0.gposts_boards (g_p_id)
-- :next
CREATE INDEX
	ON ib0.gposts_boards (b_name,g_p_id)


-- :next
CREATE TABLE ib0.boards (
	b_id      INTEGER  GENERATED ALWAYS AS IDENTITY, -- internal board ID
	b_name    TEXT     COLLATE "C",                  -- board name. if NULL, don't show as board
	newsgroup TEXT     COLLATE "C",                  -- newsgroup name. if NULL, don't expose over NNTP
	last_id   BIGINT   DEFAULT 0    NOT NULL,        -- used for post/thread IDs

	t_count BIGINT  DEFAULT 0  NOT NULL, -- thread count
	p_count BIGINT  DEFAULT 0  NOT NULL, -- post count

	badded TIMESTAMP  WITH TIME ZONE  NOT NULL, -- date added to our node
	bdesc  TEXT                       NOT NULL, -- short description

	threads_per_page INTEGER, -- <=0 - infinite, this results in only single page
	max_active_pages INTEGER, -- <=0 - all existing pages are active
	max_pages        INTEGER, -- <=0 - unlimited, archive mode

	cfg_t_bump_limit   INTEGER, -- bump limit, can be NULL
	cfg_t_thread_limit BIGINT,  -- thread limit, can be NULL

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
-- for UI-visible board list
CREATE INDEX
	ON ib0.boards (b_name COLLATE "und-x-icu")
	WHERE b_name IS NOT NULL
-- for netnews-visible grouplist
CREATE INDEX
	ON ib0.boards (newsgroup COLLATE "und-x-icu")
	WHERE newsgroup IS NOT NULL


-- :next
CREATE TABLE ib0.threads (
	b_id     INTEGER               NOT NULL, -- internal board ID this thread belongs to
	b_t_id   BIGINT                NOT NULL, -- internal board-local thread ID (ID of board-local OP post)
	g_t_id   BIGINT                NOT NULL, -- internal global thread OP post ID
	b_t_name TEXT     COLLATE "C"  NOT NULL, -- external thread identifier

	bump      TIMESTAMP  WITH TIME ZONE  NOT NULL, -- last bump time. decides position in pages/catalog
	t_order   BIGINT                     NOT NULL, -- order within board its in
	skip_over BOOLEAN                    NOT NULL, -- if true, do not include in overboard
	p_count   BIGINT                     NOT NULL DEFAULT 0, -- post count (including OP)
	f_count   BIGINT                     NOT NULL DEFAULT 0, -- sum of posts' (including OP) f_count
	fr_count  BIGINT                     NOT NULL DEFAULT 0, -- file-replies count (not including OP)

	reply_limits JSONB, -- inherits from reply_limits of ib0.boards
	thread_opts  JSONB, -- inherits from thread_opts of ib0.boards
	attrib       JSONB, -- extra attributes


	PRIMARY KEY (b_id,b_t_id),
	UNIQUE      (b_id,b_t_name),

	FOREIGN KEY (b_id)
		REFERENCES ib0.boards
		ON UPDATE CASCADE
)
-- :next
-- for board pages and catalog
CREATE INDEX
	ON ib0.threads (
		b_id   ASC,
		bump   DESC,
		b_t_id ASC
	)
-- :next
-- for overboard
CREATE INDEX
	ON ib0.threads (
		bump   DESC,
		g_t_id ASC,
		b_id   ASC
	)
	WHERE
		skip_over IS NOT TRUE


-- :next
CREATE TABLE ib0.bposts (
	b_id   INTEGER               NOT NULL, -- internal board ID this post belongs to
	b_p_id BIGINT                NOT NULL, -- internal post ID of this post. if pid==tid then this is OP
	msgid  TEXT     COLLATE "C"  NOT NULL, -- global external msgid
	p_name TEXT     COLLATE "C",           -- external post identifier
	b_t_id BIGINT,                         -- internal thread ID this post belongs to
	g_p_id BIGINT,                         -- global internal post ID

	-- denormalized w/ global for efficient indexing
	date_sent TIMESTAMP  WITH TIME ZONE,
	date_recv TIMESTAMP  WITH TIME ZONE,
	sage      BOOLEAN    NOT NULL DEFAULT FALSE, -- if true this isn't bump
	f_count   BIGINT,

	-- attributes associated with board post and visible in webui
	attrib JSON,
	-- active references
	activ_refs JSON,


	-- following fields are only used if this is mod msg
	mod_id BIGINT,
	-- used/wanted capabilities
	mod_u_cap  BIT(12),
	mod_w_cap  BIT(12),
	mod_u_bcap JSONB,
	mod_w_bcap JSONB,
	-- used(effective) cap lvls [we can't know wanted]
	mod_u_caplvl  SMALLINT ARRAY,
	mod_u_bcaplvl JSONB,
	-- self-defensive calculate effective for this board
	mod_dpriv     BIGINT,


	-- placeholder stuff
	has_ph BOOLEAN,

	ph_ban     BOOLEAN,
	ph_banpriv SMALLINT,


	PRIMARY KEY (b_id,b_p_id),
	UNIQUE      (g_p_id,b_id),

	FOREIGN KEY (b_id)
		REFERENCES ib0.boards
		ON UPDATE CASCADE,
	FOREIGN KEY (b_id,b_t_id)
		REFERENCES ib0.threads
		ON DELETE CASCADE
		ON UPDATE CASCADE,
	FOREIGN KEY (g_p_id)
		REFERENCES ib0.gposts
		ON DELETE CASCADE
		ON UPDATE CASCADE,
	FOREIGN KEY (mod_id)
		REFERENCES ib0.modlist
		ON DELETE RESTRICT
		ON UPDATE CASCADE
)
-- :next
-- in thread, for bump
CREATE INDEX
	ON ib0.bposts (
		b_id      ASC,
		b_t_id    ASC,
		date_sent ASC,
		b_p_id    ASC
	)
-- :next
-- for NEWNEWS (yeh, not in ib0.gposts)
CREATE INDEX
	ON ib0.bposts (
		date_recv,
		g_p_id,
		b_id
	)
-- :next
-- for post num lookup
CREATE UNIQUE INDEX
	ON ib0.bposts (
		p_name text_pattern_ops,
		b_id
	)
-- :next
-- mostly for boardban checks for now
CREATE UNIQUE INDEX
	ON ib0.bposts (
		msgid,
		b_id
	)
-- :next
-- FK
CREATE INDEX
	ON ib0.bposts (
		mod_id,
		date_sent
	)
	WHERE mod_id IS NOT NULL


-- :next
-- if OP gets nuked we want to nuke whole thread
-- NOTE: OP may get replaced with ban
ALTER TABLE
	ib0.threads
ADD CONSTRAINT
	fk_bposts
FOREIGN KEY (b_id,b_t_id)
	REFERENCES ib0.bposts (b_id,b_p_id)
	MATCH FULL
	ON DELETE CASCADE
	ON UPDATE CASCADE


-- :next
CREATE TYPE ftype_t AS ENUM (
	'file',  -- normal unknown/fallback
	'msg',   -- original message
	'face',  -- decoded X-Face / Face hdr
	'text',  -- .txt file
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
		REFERENCES ib0.gposts
		ON DELETE CASCADE
		ON UPDATE CASCADE
)
-- :next
-- "lookup by g_p_id"
-- f_id helps sorted retrieval
CREATE INDEX ON ib0.files (g_p_id,f_id)
-- :next
-- XXX is this used by something?
CREATE INDEX ON ib0.files (fname,thumb)


-- :next
-- for fnames GC
CREATE TABLE ib0.files_uniq_fname (
	-- key
	fname  TEXT    COLLATE "C"  NOT NULL,

	-- count
	cnt    BIGINT               NOT NULL,


	PRIMARY KEY (fname)
)
-- :next
-- for thumbs GC
CREATE TABLE ib0.files_uniq_thumb (
	-- key
	fname  TEXT    COLLATE "C"  NOT NULL,
	thumb  TEXT    COLLATE "C"  NOT NULL,

	-- count
	cnt    BIGINT               NOT NULL,


	PRIMARY KEY (fname,thumb)
)

-- :next
-- distinct capability grants
-- these would be deleted/reinserted if priv of mod behind them changes
CREATE TABLE ib0.modsets (
	mod_pubkey TEXT COLLATE "C" NOT NULL,
	-- if limited to single group
	mod_group  TEXT COLLATE "C",
	-- usable
	mod_cap    BIT(12)          NOT NULL,
	mod_caplvl SMALLINT ARRAY,
	-- inheritable
	modi_cap    BIT(12)         NOT NULL,
	modi_caplvl SMALLINT ARRAY,
	-- board post responsible for this modset (if any)
	b_id     INTEGER,
	b_p_id   BIGINT,

	FOREIGN KEY (b_id,b_p_id)
		REFERENCES ib0.bposts
		MATCH FULL
		ON DELETE CASCADE    -- trigger corrects stuff
)
-- :next
CREATE UNIQUE INDEX
	ON ib0.modsets (mod_pubkey)
	WHERE
		b_id IS NULL AND
			b_p_id IS NULL AND
			mod_group IS NULL
-- :next
CREATE UNIQUE INDEX
	ON ib0.modsets (mod_pubkey,mod_group)
	WHERE
		b_id IS NULL AND
			b_p_id IS NULL AND
			mod_group IS NOT NULL


-- :next
-- refers-refered relation. used only to awaken re-calculation.
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


-- :next
CREATE TABLE ib0.banlist (
	ban_id   BIGINT GENERATED ALWAYS AS IDENTITY,
	ban_info TEXT   NOT NULL,

	-- board post responsible for this ban (if any)
	b_id     INTEGER,
	b_p_id   BIGINT,
	-- del power of this ban
	dpriv    SMALLINT,

	-- msgid being banned
	msgid  TEXT  COLLATE "C" NOT NULL,
	-- if per-board ban [board may not exist yet therefore TEXT]
	b_name  TEXT  COLLATE "C",


	PRIMARY KEY (ban_id),

	FOREIGN KEY (b_id,b_p_id)
		REFERENCES ib0.bposts
		MATCH FULL
		ON DELETE CASCADE    -- see trigger below
)
-- :next
-- for foreign key, efficient delete
CREATE INDEX
	ON ib0.banlist (b_id,b_p_id)
	WHERE b_id IS NOT NULL AND b_p_id IS NOT NULL
-- :next
-- for non-board ban aggregation
CREATE INDEX
	ON ib0.banlist (msgid)
	WHERE b_name IS NULL
-- :next
-- for board ban aggregation
-- TODO idk how exactly they'll be aggregated yet so bit early
CREATE INDEX
	ON ib0.banlist (b_name,msgid)
	WHERE b_name IS NOT NULL
