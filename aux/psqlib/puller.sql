-- :name puller_get_last_newnews
SELECT
	last_newnews
FROM
	ib0.puller_last_newnews
WHERE
	sid = $1

-- :name puller_set_last_newnews
INSERT INTO
	ib0.puller_last_newnews AS ln (sid,last_newnews)
VALUES
	($1,$2)
ON CONFLICT
	(sid)
DO
	UPDATE SET
		last_newnews = $2
	WHERE
		ln.sid = $1

-- :name puller_get_last_newsgroups
SELECT
	last_newgroups
FROM
	ib0.puller_last_newgroups
WHERE
	sid = $1

-- :name puller_set_last_newsgroups
INSERT INTO
	ib0.puller_last_newgroups AS ln (sid,last_newgroups)
VALUES
	($1,$2)
ON CONFLICT
	(sid)
DO
	UPDATE SET
		last_newgroups = $2
	WHERE
		ln.sid = $1

-- :name puller_get_group_id
SELECT
	sg.b_id,
	st.last_max
FROM
	ib0.boards AS sg
LEFT JOIN LATERAL
	(
		SELECT
			xt.last_max AS last_max
		FROM
			ib0.puller_group_track xt
		WHERE
			xt.b_id = sg.b_id AND xt.sid = $1
	) AS st
ON
	TRUE
WHERE
	sg.b_name = $2

-- :name puller_set_group_id
UPDATE
	ib0.puller_group_track AS st
SET
	last_max = $3
FROM
	ib0.boards AS xb
WHERE
	st.sid=$1 AND xb.b_name=$2 AND st.bid=xb.b_id

-- :name puller_unset_group_id
DELETE FROM
	ib0.puller_group_track AS st
USING
	ib0.boards xb
WHERE
	st.sid=$1 AND xb.b_name=$2 AND st.bid=xb.b_id

-- :name puller_load_temp_groups
SELECT
	xb.b_name,
	xs.next_max,
	xs.last_max
FROM
	ib0.puller_group_track xs
JOIN
	ib0.boards xb
ON
	xs.bid = xb.b_id
WHERE
	xs.sid=$1 AND xs.last_use=$2
ORDER BY
	xb.b_name
