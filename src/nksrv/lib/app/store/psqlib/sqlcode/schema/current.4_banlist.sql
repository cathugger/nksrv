CREATE TABLE ib.banlist (
	ban_id   BIGINT GENERATED ALWAYS AS IDENTITY,
	ban_info TEXT   NOT NULL,

	-- board post responsible for this ban (if any)
	b_id     INTEGER,
	b_p_id   BIGINT,
	-- del power of this ban
	dpriv    SMALLINT,

    -- ban target
	bt_msgid  TEXT  COLLATE "C" NOT NULL, -- msgid being banned
	bt_b_id   INTEGER,                    -- if per-board ban


	PRIMARY KEY (ban_id),

	FOREIGN KEY (b_id,b_p_id)
		REFERENCES ib.bposts
		MATCH FULL
		ON DELETE CASCADE,  -- see trigger below

    FOREIGN KEY (bt_b_id)
        REFERENCES ib.boards
        ON DELETE RESTRICT
        ON UPDATE CASCADE
);

-- for foreign key, efficient delete
CREATE INDEX
	ON ib.banlist (
        b_id,
        b_p_id
    )
	WHERE
        b_id IS NOT NULL AND
            b_p_id IS NOT NULL;

-- for non-board ban aggregation
CREATE INDEX
	ON ib.banlist (
        bt_msgid
    )
	WHERE
        bt_b_id IS NULL;

-- for board ban aggregation
CREATE INDEX
	ON ib.banlist (
        bt_b_id,
        bt_msgid
    )
	WHERE
        bt_b_id IS NOT NULL;
