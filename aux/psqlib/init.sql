--- core stuff
-- :name version
demo8
-- :name init
CREATE SCHEMA ib0


--- base stuff


--- moderators/administrators things
-- :next
-- summary table to hold effective capabilities of moderator
CREATE TABLE ib0.modlist (
	mod_id     BIGINT     GENERATED ALWAYS AS IDENTITY,
	mod_pubkey TEXT       COLLATE "C"  NOT NULL,
	mod_name   TEXT,
	-- if true, then no modpriv is holding it [so can be GC'd]
	automanage BOOLEAN  NOT NULL,
	mod_cap    BIT(2),   -- global capabilities
	mod_bcap   JSONB,    -- per-board capabilities
	mod_dpriv  SMALLINT, -- global delete privilege
	mod_bdpriv JSONB,    -- per-board delete privileges

	PRIMARY KEY (mod_id),
	UNIQUE      (mod_pubkey)
)

-- :next
-- table to log changes of modlist
-- used to keep state for mod msg reprocessing
CREATE TABLE ib0.modlist_changes (
	mlc_id BIGINT NOT NULL,
	mod_id BIGINT NOT NULL,
	-- tracked state
	t_pdate  TIMESTAMP WITH TIME ZONE,
	t_g_p_id BIGINT,
	t_b_id   INTEGER,

	PRIMARY KEY (mlc_id),
	UNIQUE      (mod_id),
	FOREIGN KEY (mod_id)
		REFERENCES ib0.modlist
		ON CASCADE DELETE
)

-- :next
CREATE FUNCTION
	ib0.modlist_changepriv()
RETURNS
	TRIGGER
AS
$$
BEGIN

	INSERT INTO
		ib0.modlist_changes (
			mod_id,
			t_pdate,
			t_g_p_id,
			t_b_id
		)
	VALUES
		(
			NEW.mod_id,
			NULL,
			NULL,
			NULL
		)
	ON CONFLICT
		(mod_id)
	DO UPDATE
		SET
			t_pdate  = EXCLUDED.t_pdate,
			t_g_p_id = EXCLUDED.t_g_p_id,
			t_b_id   = EXCLUDED.t_b_id

	-- poke process which can act upon it
	NOTIFY ib0_modlist_changes;

	RETURN NULL;

END;
$$
LANGUAGE
	plpgsql

-- :next
-- if delete, then there are no posts to invoke by now
-- if insert, then there are no posts to invoke yet
CREATE TRIGGER
	modlist_changepriv
AFTER
	UPDATE OF
		mod_cap,
		mod_bcap,
		mod_dpriv,
		mod_bdpriv
ON
	ib0.modsets
FOR EACH
	ROW
WHEN
	(OLD.mod_cap,OLD.mod_bcap,OLD.mod_dpriv,OLD.mod_bdpriv) IS DISTINCT FROM
		(NEW.mod_cap,NEW.mod_bcap,NEW.mod_dpriv,NEW.mod_bdpriv)
EXECUTE PROCEDURE
	ib0.modlist_changepriv()






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

	-- for ban placeholders in ctl groups
	ban_dpriv SMALLINT,

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
	fr_count  BIGINT                     NOT NULL, -- file-replies count (not including OP)

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
	pdate  TIMESTAMP  WITH TIME ZONE,           -- real date field
	padded TIMESTAMP  WITH TIME ZONE,           -- date field used for sorting. will actually contain delivery date
	sage   BOOLEAN                    NOT NULL, -- if true this isn't bump

	-- following fields are only used if this is mod msg
	mod_id BIGINT,
	-- used/wanted capabilities
	mod_u_cap  BIT(2),
	mod_w_cap  BIT(2),
	mod_u_bcap JSONB,
	mod_w_bcap JSONB,
	-- used(effective) dprivs [we can't know wanted]
	mod_u_dpriv  SMALLINT,
	mod_u_bdpriv JSONB,

	-- if this is ban placeholder, which priv lvl it'd need to break?
	ban_dpriv SMALLINT,

	-- attributes associated with board post and visible in webui
	attrib JSON,
	-- active references
	activ_refs JSON,

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
-- distinct capability grants
CREATE TABLE ib0.modsets (
	mod_pubkey TEXT COLLATE "C" NOT NULL,
	mod_cap    BIT(2)           NOT NULL,
	mod_dpriv  SMALLINT,
	mod_group  TEXT COLLATE "C",
	-- board post responsible for this ban (if any)
	b_id     INTEGER,
	b_p_id   BIGINT,

	FOREIGN KEY (b_id,b_p_id)
		REFERENCES ib0.bposts (b_id,b_p_id)
		MATCH FULL
		ON DELETE CASCADE    -- see trigger below
)
-- :next
-- to be ran AFTER delet from modsets
CREATE FUNCTION
	ib0.modsets_compute()
RETURNS
	TRIGGER
AS
$$
DECLARE
	pubkey   TEXT;
	r        RECORD;
	u_mod_id BIGINT;
BEGIN
	-- setup pubkey var
	IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
		pubkey := NEW.mod_pubkey;
	ELSIF TG_OP = 'DELETE' THEN
		pubkey := OLD.mod_pubkey;
	END;
	-- recalc modlist val from modsets
	WITH
		comp_caps AS (
			SELECT
				mod_group,
				bit_or(mod_priv) AS mod_calcpriv
			FROM
				ib0.modsets
			WHERE
				mod_pubkey = pubkey
			GROUP BY
				mod_group
			ORDER BY
				mod_group
		)
	SELECT
		x.mod_priv,
		y.mod_bpriv,
		z.automanage
	INTO STRICT
		r
	FROM
		(
			SELECT
				mod_calcpriv AS mod_priv
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS x,
		(
			SELECT
				jsonb_object(
					array_agg(mod_group),
					array_agg(mod_calcpriv::TEXT)) AS mod_bpriv
			FROM
				comp_caps
			WHERE
				mod_group IS NOT NULL
		) AS y,
		(
			SELECT
				COUNT(*) = 0 AS automanage
			FROM
				comp_caps
		) AS z;

	IF TG_OP = 'INSERT' THEN
		-- insert or update
		-- it may not exist yet
		-- or it may be automanaged
		INSERT INTO
			ib0.modlist (
				mod_pubkey,
				mod_priv,
				mod_bpriv,
				automanage
			)
		VALUES (
			pubkey,
			r.mod_priv,
			r.mod_bpriv,
			r.automanage
		)
		ON CONFLICT
			(mod_pubkey)
		DO UPDATE
			SET
				mod_priv   = EXCLUDED.mod_priv,
				mod_bpriv  = EXCLUDED.mod_bpriv,
				automanage = EXCLUDED.automanage;

	ELSIF TG_OP = 'UPDATE' THEN
		-- only update existing (because at this point it will exist)
		-- at this point it'll be automanaged too (because we're moding existing row)
		UPDATE
			ib0.modlist
		SET
			mod_priv  = r.mod_priv,
			mod_bpriv = r.mod_bpriv
		WHERE
			mod_pubkey = pubkey;

	ELSIF TG_OP = 'DELETE' THEN
		-- update and possibly delete
		UPDATE
			ib0.modlist
		SET
			mod_priv   = r.mod_priv,
			mod_bpriv  = r.mod_bpriv,
			automanage = r.automanage
		WHERE
			mod_pubkey = pubkey
		RETURNING
			mod_id
		INTO STRICT
			u_mod_id;

		IF r.automanage THEN
			-- if it's automanaged, do GC incase no post refers to it
			DELETE FROM
				ib0.modlist mods
			USING
				(
					SELECT
						mod_id,
						COUNT(*) <> 0 AS hasrefs
					FROM
						ib0.bposts
					WHERE
						mod_id = u_mod_id
					GROUP BY
						mod_id
				) AS rcnts
			WHERE
				mods.mod_id = rcnts.mod_id AND
					rcnts.hasrefs = FALSE
		END IF;

	END IF;

	RETURN NULL;
END;
$$
LANGUAGE
	plpgsql

-- :next
CREATE TRIGGER
	modsets_compute
AFTER
	INSERT OR UPDATE OR DELETE
ON
	ib0.modsets
FOR EACH
	ROW
EXECUTE PROCEDURE
	ib0.modsets_compute()







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
		REFERENCES ib0.bposts (b_id,b_p_id)
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
	ib0.banlist_after_del()
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
	banlist_after_del
AFTER
	DELETE
ON
	ib0.banlist
REFERENCING
	OLD TABLE AS oldrows
FOR EACH
	STATEMENT
EXECUTE PROCEDURE
	ib0.banlist_after_del()
