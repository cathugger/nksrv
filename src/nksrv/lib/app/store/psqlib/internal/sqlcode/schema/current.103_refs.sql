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
);
-- foreign key
CREATE INDEX
	ON ib.refs (
        b_id,
        b_p_id
    );
-- postname x boardname
CREATE INDEX
	ON ib.refs (
        p_name text_pattern_ops,
        b_name NULLS FIRST
    )
	WHERE
        p_name IS NOT NULL;
-- boardname
CREATE INDEX
	ON ib.refs (
        b_name
    )
	WHERE
        b_name IS NOT NULL AND p_name IS NULL;
-- pure msgid refs
CREATE INDEX
	ON ib.refs (
        msgid
    )
	WHERE
        msgid IS NOT NULL;
