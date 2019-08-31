-- modification related queries

-- :name mod_ref_write
-- NOTE: this used to track failed references, therefore needed update
-- it used to contain delete stmt, but since we now track all refs,
-- it's no longer needed
INSERT INTO
	ib0.failrefs (
		b_id,
		b_p_id,
		p_name,
		b_name,
		msgid
	)
SELECT
	$1,
	$2,
	unnest($3::text[]) AS p_name,
	unnest($4::text[]) AS b_name,
	unnest($5::text[]) AS msgid

-- :name mod_ref_find_post
-- args: off_b,off_b_p,p_name,board,msgid
WITH
	msgs AS (
		SELECT
			*
		FROM
			(
				SELECT
					b_id,b_p_id
				FROM
					ib0.refs
				WHERE
					-- index-search by first 8 bytes, then narrow
					(p_name LIKE substring($3 for 8) || '%') AND
						($3 LIKE p_name || '%') AND
						(b_name IS NULL OR b_name = $4)
				UNION
				SELECT
					b_id,b_p_id
				FROM
					ib0.refs
				WHERE
					msgid = $5
				ORDER BY
					b_id,b_p_id
			) AS x
		WHERE
			-- this will probably be inefficient
			-- but considering that we're assembling info from multiple indexes,
			-- I think it's okay
			(b_id,b_p_id) > ($1,$2)
		LIMIT
			5000
	)
SELECT
	xbp.b_id,
	xbp.b_p_id,
	xp.message,
	xp.headers -> 'In-Reply-To' ->> 0,
	xbp.attrib,
	xbp.t_id
FROM
	msgs
JOIN
	ib0.bposts AS xbp
ON
	xbp.b_id = msgs.b_id AND xbp.b_p_id = msgs.b_p_id
JOIN
	ib0.posts AS xp
ON
	xp.g_p_id = xbp.g_p_id

-- :name mod_update_bpost_attrib
UPDATE
	ib0.bposts
SET
	attrib = $3
WHERE
	(b_id,b_p_id) = ($1,$2)
