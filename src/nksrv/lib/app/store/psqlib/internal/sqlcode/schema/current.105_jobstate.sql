
-- table to log changes of modlist
-- used to keep state for mod msg reprocessing
CREATE TABLE ib.modlist_changes (

	j_id   BIGINT   GENERATED ALWAYS AS IDENTITY   PRIMARY KEY,

	-- what caused this change
	mod_id   BIGINT   NOT NULL,

	-- change state
	t_date_sent TIMESTAMP WITH TIME ZONE,
	t_g_p_id    BIGINT,
	t_b_id      INTEGER,


	FOREIGN KEY (mod_id)
		REFERENCES ib.modlist
		ON DELETE CASCADE
);

CREATE INDEX
    ON ib.modlist_changes (
        mod_id
    );



-- table used to hold state of bpost backrefs processing
-- bpost backrefs are processed from the bottom to the top
CREATE TABLE ib.refs_deps_recalc (

	j_id   BIGINT   GENERATED ALWAYS AS IDENTITY   PRIMARY KEY,

	-- what caused this change
	p_name TEXT  COLLATE "C",
	b_name TEXT  COLLATE "C",
	msgid  TEXT  COLLATE "C",

	-- change state
	b_id   INTEGER,
	b_p_id BIGINT
);



-- table used to hold bpost ids TODO'd for recalc
CREATE TABLE ib.refs_recalc (

	j_id   BIGINT   GENERATED ALWAYS AS IDENTITY   PRIMARY KEY,

	b_id   INTEGER NOT NULL,
	b_p_id BIGINT  NOT NULL,


	FOREIGN KEY (b_id,b_p_id)
		REFERENCES ib.bposts
		MATCH FULL
		ON DELETE CASCADE
);

CREATE INDEX
    ON ib.refs_recalc (
        b_id,
        b_p_id
    );


-- table used to hold files which should be deleted
-- we can't delete files before tx is successfuly commited (otherwise if commit fails/crashed, we gonna be in broken state),
-- and after it's committed we MUST hold lock on counter before delete to prevent nuking of new files
-- XXX we could nuke file/thumb counter tables and rely on serialized mode's siread locks; how would performance compare?
-- a bit of backlog aint going to cause any troubles
CREATE TABLE ib.files_deleted (
	d_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	fname TEXT NOT NULL
);

-- same for file thumbnails
CREATE TABLE ib.fthumbs_deleted (
	d_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	fname TEXT NOT NULL,
	thumb TEXT NOT NULL
);
