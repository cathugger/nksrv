-- summary table to hold effective aggregated capabilities of moderator
CREATE TABLE ib.modlist (
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
);

ALTER TABLE
	ib.bposts
ADD CONSTRAINT
	fk_bposts_mod_id
FOREIGN KEY (mod_id)
	REFERENCES ib.modlist
	ON DELETE RESTRICT
	ON UPDATE CASCADE;



-- distinct capability grants
-- these would be deleted/reinserted if priv of mod behind them changes
CREATE TABLE ib.modsets (
	mod_pubkey TEXT COLLATE "C" NOT NULL,
	-- if limited to single newsgroup
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
);

CREATE UNIQUE INDEX
	ON ib.modsets (mod_pubkey)
	WHERE
		b_id IS NULL AND
			b_p_id IS NULL AND
			mod_group IS NULL;

CREATE UNIQUE INDEX
	ON ib.modsets (mod_pubkey,mod_group)
	WHERE
		b_id IS NULL AND
			b_p_id IS NULL AND
			mod_group IS NOT NULL;

