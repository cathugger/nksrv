
-- boards/newsgroups
CREATE TABLE ib.boards (

	b_id      INTEGER  GENERATED ALWAYS AS IDENTITY, -- internal board ID
	b_webname TEXT     COLLATE "und-x-icu",          -- board name. if NULL, don't expose on the web
	newsgroup TEXT     COLLATE "und-x-icu",          -- newsgroup name. if NULL, don't expose over NNTP

	b_added TIMESTAMP  WITH TIME ZONE  NOT NULL, -- date added to our node
	b_desc  TEXT                       NOT NULL, -- short description

	last_id BIGINT  NOT NULL  DEFAULT 0, -- used for post/thread IDs XXX separate?

	c_t_pos BIGINT NOT NULL DEFAULT 0,
    c_t_neg BIGINT NOT NULL DEFAULT 0,
    c_p_pos BIGINT NOT NULL DEFAULT 0,
    c_p_neg BIGINT NOT NULL DEFAULT 0,
    c_f_pos BIGINT NOT NULL DEFAULT 0,
    c_f_neg BIGINT NOT NULL DEFAULT 0,
    c_d_pos BIGINT NOT NULL DEFAULT 0,
    c_d_neg BIGINT NOT NULL DEFAULT 0,

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


	PRIMARY KEY (b_id)
);
-- for NEWGROUPS
CREATE INDEX
	ON ib.boards (badded,b_id);
-- for UI-visible board list
CREATE UNIQUE INDEX
	ON ib.boards (b_webname COLLATE "und-x-icu")
	WHERE b_webname IS NOT NULL;
-- for netnews-visible grouplist
CREATE UNIQUE INDEX
	ON ib.boards (newsgroup COLLATE "und-x-icu")
	WHERE newsgroup IS NOT NULL;



CREATE TABLE ib.threads (

	b_id     INTEGER               NOT NULL, -- internal board ID this thread belongs to
	b_t_id   BIGINT                NOT NULL, -- internal board-local thread ID (ID of board-local OP post)
	g_t_id   BIGINT                NOT NULL, -- internal global thread OP post ID
	b_t_name TEXT     COLLATE "C"  NOT NULL, -- external thread identifier

	bump TIMESTAMP  WITH TIME ZONE  NOT NULL, -- last bump time. decides position in pages/catalog

	t_arch_order BIGINT  NOT NULL, -- order number within arcive (newest thread goes last, so don't need updating of old indexes)

	skip_over BOOLEAN  NOT NULL,            -- if true, do not include in overboard
	p_count   BIGINT   NOT NULL  DEFAULT 0, -- post count (including OP)
	f_count   BIGINT   NOT NULL  DEFAULT 0, -- sum of posts' (including OP) f_count
	fr_count  BIGINT   NOT NULL  DEFAULT 0, -- file-replies (replies containing files) count (not including OP)

	reply_limits JSONB, -- inherits from reply_limits of ib.boards
	thread_opts  JSONB, -- inherits from thread_opts of ib.boards
	attrib       JSONB, -- extra attributes


	PRIMARY KEY (b_id,b_t_id),
	UNIQUE      (b_id,b_t_name),

	FOREIGN KEY (b_id)
		REFERENCES ib.boards
		ON UPDATE CASCADE
);
-- for board pages and catalog
CREATE INDEX
	ON ib.threads (
		b_id   ASC,
		bump   DESC,
		b_t_id ASC
	);
-- for overboard
CREATE INDEX
	ON ib.threads (
		bump   DESC,
		g_t_id ASC,
		b_id   ASC
	)
	WHERE
		skip_over IS NOT TRUE;




-- main posts table
CREATE TABLE ib.gposts (
	g_p_id BIGINT  GENERATED ALWAYS AS IDENTITY, -- global internal post ID
	msgid  TEXT    COLLATE "C"  NOT NULL,        -- Message-ID

	date_sent TIMESTAMP  WITH TIME ZONE,
	date_recv TIMESTAMP  WITH TIME ZONE,
	sage      BOOLEAN    NOT NULL        DEFAULT FALSE,

	f_count INTEGER  NOT NULL  DEFAULT 0, -- attachment count
	f_size  BIGINT   NOT NULL  DEFAULT 0, -- total attachment size in storage

	author  TEXT               NOT NULL  DEFAULT '', -- author name
	trip    TEXT  COLLATE "C"  NOT NULL  DEFAULT '', -- XXX should we have it there and not in attrib? probably yes, we could benefit from search
	title   TEXT               NOT NULL  DEFAULT '', -- message title/subject field
	body    TEXT               NOT NULL  DEFAULT '', -- post body, in UTF-8

	headers JSONB, -- headers of msg root, map of lists of strings, needed for NNTP HDR
	attrib  JSON,  -- attributes associated with global post and visible in webui
	layout  JSON,  -- article layout, needed to reconstruct original article
	extras  JSONB, -- passive extra data

	mod_dpriv SMALLINT, -- calc'd from bposts

	-- does it have placeholder-related data?
	has_ph BOOLEAN GENERATED ALWAYS AS (ph_ban IS NOT NULL) STORED,

	ph_ban     BOOLEAN,
	ph_banpriv SMALLINT,


	PRIMARY KEY (g_p_id),
	UNIQUE      (msgid)
);



CREATE TABLE ib.bposts (
	b_id   INTEGER               NOT NULL, -- internal board ID this post belongs to
	b_p_id BIGINT                NOT NULL, -- internal post ID of this post. if pid==tid then this is OP
	msgid  TEXT     COLLATE "C",           -- global external msgid
	p_name TEXT     COLLATE "C",           -- external post identifier
	b_t_id BIGINT,                         -- internal thread ID this post belongs to
	g_p_id BIGINT,                         -- global internal post ID

	-- denormalized w/ global for efficient indexing
	date_sent TIMESTAMP  WITH TIME ZONE,
	date_recv TIMESTAMP  WITH TIME ZONE,
	sage      BOOLEAN    NOT NULL        DEFAULT FALSE, -- if true this isn't bump

	f_count INTEGER  NOT NULL  DEFAULT 0,
	f_size  BIGINT   NOT NULL  DEFAULT 0,

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
		REFERENCES ib.boards
		ON UPDATE CASCADE,
	FOREIGN KEY (b_id,b_t_id)
		REFERENCES ib.threads
		ON DELETE CASCADE
		ON UPDATE CASCADE,
	FOREIGN KEY (g_p_id)
		REFERENCES ib.gposts
		ON DELETE CASCADE
		ON UPDATE CASCADE,
	FOREIGN KEY (mod_id)
		REFERENCES ib.modlist
		ON DELETE RESTRICT
		ON UPDATE CASCADE
);
