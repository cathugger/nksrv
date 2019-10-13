-- :name init_jobstate

-- :next
-- table to log changes of modlist
-- used to keep state for mod msg reprocessing
CREATE TABLE ib0.modlist_changes (
	mlc_id BIGINT NOT NULL,
	mod_id BIGINT NOT NULL,
	-- tracked state
	t_date_sent TIMESTAMP WITH TIME ZONE,
	t_g_p_id    BIGINT,
	t_b_id      INTEGER,

	PRIMARY KEY (mlc_id),
	UNIQUE      (mod_id),
	FOREIGN KEY (mod_id)
		REFERENCES ib0.modlist
		ON DELETE CASCADE
)

-- :next
-- table used to hold state of bpost ref recalculations
-- bpost refs are recalc'd from the bottom to the top
CREATE TABLE ib0.refs_recalc (
	j_id   BIGINT  GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

	b_id   INTEGER NOT NULL,
	b_p_id BIGINT  NOT NULL,

	p_name TEXT  COLLATE "C",
	b_name TEXT  COLLATE "C",
	msgid  TEXT  COLLATE "C"
)
