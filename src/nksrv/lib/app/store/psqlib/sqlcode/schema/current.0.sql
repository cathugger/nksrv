-- :set version 0.1.0
CREATE SCHEMA ib;








-- :next
-- :next
-- in thread, for bump
CREATE INDEX
	ON ib.bposts (
		b_id      ASC,
		b_t_id    ASC,
		date_sent ASC,
		b_p_id    ASC
	)
-- :next
-- for NEWNEWS (yeh, not in ib.gposts)
CREATE INDEX
	ON ib.bposts (
		date_recv,
		g_p_id,
		b_id
	)
-- :next
-- for post num lookup
CREATE UNIQUE INDEX
	ON ib.bposts (
		p_name text_pattern_ops,
		b_id
	)
-- :next
-- mostly for boardban checks for now
CREATE UNIQUE INDEX
	ON ib.bposts (
		msgid,
		b_id
	)
-- :next
-- FK
CREATE INDEX
	ON ib.bposts (
		mod_id,
		date_sent
	)
	WHERE mod_id IS NOT NULL


-- :next
-- if OP gets nuked we want to nuke whole thread
-- NOTE: OP may get replaced with ban
ALTER TABLE
	ib.threads
ADD CONSTRAINT
	fk_bposts
FOREIGN KEY (b_id,b_t_id)
	REFERENCES ib.bposts (b_id,b_p_id)
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
CREATE TABLE ib.files (
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
		REFERENCES ib.gposts
		ON DELETE CASCADE
		ON UPDATE CASCADE
)
-- :next
-- "lookup by g_p_id"
-- f_id helps sorted retrieval
CREATE INDEX ON ib.files (g_p_id,f_id)
-- :next
-- XXX is this used by something?
CREATE INDEX ON ib.files (fname,thumb)


-- :next
-- for fnames GC
CREATE TABLE ib.files_uniq_fname (
	-- key
	fname  TEXT    COLLATE "C"  NOT NULL,

	-- count
	cnt    BIGINT               NOT NULL,


	PRIMARY KEY (fname)
)
-- :next
-- for thumbs GC
CREATE TABLE ib.files_uniq_thumb (
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
CREATE TABLE ib.modsets (
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
		REFERENCES ib.bposts
		MATCH FULL
		ON DELETE CASCADE    -- trigger corrects stuff
)
-- :next
CREATE UNIQUE INDEX
	ON ib.modsets (mod_pubkey)
	WHERE
		b_id IS NULL AND
			b_p_id IS NULL AND
			mod_group IS NULL
-- :next
CREATE UNIQUE INDEX
	ON ib.modsets (mod_pubkey,mod_group)
	WHERE
		b_id IS NULL AND
			b_p_id IS NULL AND
			mod_group IS NOT NULL


-- :next
-- refers-refered relation. used only to awaken re-calculation.
-- references are rendered per-board, not per-post,
-- as multiboard posts may end up refering to odd things otherwise
CREATE TABLE ib.refs (
	-- board post who owns reference
	b_id   INTEGER NOT NULL,
	b_p_id BIGINT  NOT NULL,

	p_name TEXT  COLLATE "C", -- external post identifier (or part of it)
	b_name TEXT  COLLATE "C", -- board name
	msgid  TEXT  COLLATE "C", -- Message-ID

	FOREIGN KEY (b_id,b_p_id)
		REFERENCES ib.bposts
		ON DELETE CASCADE
)
-- :next
CREATE INDEX
	ON ib.refs (b_id,b_p_id) -- FK
-- :next
CREATE INDEX
	ON ib.refs (p_name text_pattern_ops,b_name NULLS FIRST)
	WHERE p_name IS NOT NULL
-- :next
CREATE INDEX
	ON ib.refs (b_name)
	WHERE b_name IS NOT NULL AND p_name IS NULL
-- :next
CREATE INDEX
	ON ib.refs (msgid)
	WHERE msgid IS NOT NULL


-- :next
CREATE TABLE ib.banlist (
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
		REFERENCES ib.bposts
		MATCH FULL
		ON DELETE CASCADE    -- see trigger below
)
-- :next
-- for foreign key, efficient delete
CREATE INDEX
	ON ib.banlist (b_id,b_p_id)
	WHERE b_id IS NOT NULL AND b_p_id IS NOT NULL
-- :next
-- for non-board ban aggregation
CREATE INDEX
	ON ib.banlist (msgid)
	WHERE b_name IS NULL
-- :next
-- for board ban aggregation
-- TODO idk how exactly they'll be aggregated yet so bit early
CREATE INDEX
	ON ib.banlist (b_name,msgid)
	WHERE b_name IS NOT NULL
