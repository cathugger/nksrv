-- :name init_puller

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
