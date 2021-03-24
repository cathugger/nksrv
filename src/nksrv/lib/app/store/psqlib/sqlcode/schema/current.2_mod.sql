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

