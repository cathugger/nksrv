-- :name nntp_article_exists_or_banned_by_msgid
-- input: cmsgid
-- output: 1
SELECT
	1
FROM
	ib.gposts
WHERE
	msgid = $1

-- :name nntp_article_valid_by_msgid
-- input: cmsgid
-- output: dummy 1
SELECT
	1
FROM
	ib.gposts
WHERE
	msgid = $1 AND date_recv IS NOT NULL


-- :name nntp_article_num_by_msgid
-- input: msgid curr_group_id
SELECT
	xbp.b_id,
	xbp.b_p_id,
	xp.g_p_id,
	(xp.date_recv IS NULL) AS is_banned
FROM
	ib.gposts AS xp
JOIN
	ib.bposts AS xbp
USING
	(g_p_id)
WHERE
	xp.msgid = $1
ORDER BY
	(xbp.b_id = $2) DESC
LIMIT
	1

-- :name nntp_article_msgid_by_num
-- input: bid bpid
SELECT
	xbp.msgid,
	xbp.g_p_id
FROM
	ib.bposts AS xbp
WHERE
	xbp.b_id = $1 AND xbp.b_p_id = $2



-- :name nntp_article_get_gpid
-- input: gpid
SELECT
	jp.title,
	jp.message,
	jp.headers,
	jp.layout,
	jf.fname,
	jf.fsize
FROM
	ib.gposts AS jp
LEFT JOIN
	ib.files AS jf
USING
	(g_p_id)
WHERE
	-- ensure g_p_id points to valid post to prevent TOCTOU hazards
	jp.g_p_id = $1 AND jp.date_recv IS NOT NULL
ORDER BY
	jf.f_id



-- :name nntp_select
-- input: {newsgroup}
WITH
	z AS (
		SELECT
			xb.b_id        AS b_id,
			xb.p_count     AS p_count,
			MIN(xp.b_p_id) AS lo,
			MAX(xp.b_p_id) AS hi
		FROM
			ib.boards AS xb
		LEFT JOIN
			ib.bposts AS xp
		USING
			(b_id)
		WHERE
			xb.newsgroup = $1
		GROUP BY
			xb.b_id
	)
SELECT
	z.b_id,
	z.p_count,
	z.lo,
	z.hi,
	zbp.g_p_id
FROM
	z
LEFT JOIN
	ib.bposts as zbp
ON
	z.b_id = zbp.b_id AND z.lo = zbp.b_p_id

-- :name nntp_select_and_list
-- input: {board name} {min} {max}
SELECT
	xbi.b_id,
	xbi.p_count,
	xbi.lo,
	xbi.hi,
	xbi.g_lo,
	x3.b_p_id
FROM
	(
		SELECT
			xbe.b_id    AS b_id,
			xbe.p_count AS p_count,
			xbe.lo      AS lo,
			xbe.hi      AS hi,
			xbp.g_p_id  AS g_lo
		FROM
			(
				SELECT
					xb.b_id         AS b_id,
					xb.p_count      AS p_count,
					MIN(xbp.b_p_id) AS lo,
					MAX(xbp.b_p_id) AS hi
				FROM
					ib.boards AS xb
				LEFT JOIN
					ib.bposts AS xbp
				USING
					(b_id)
				WHERE
					xb.newsgroup = $1
				GROUP BY
					xb.b_id
			) AS xbe
		LEFT JOIN
			ib.bposts AS xbp
		ON
			xbe.b_id = xbp.b_id AND xbe.lo = xbp.b_p_id
	) AS xbi
LEFT JOIN
	ib.bposts AS x3
ON
	xbi.b_id = x3.b_id AND
		x3.b_p_id >= $2 AND
		($3 < 0 OR x3.b_p_id <= $3)
ORDER BY
	x3.b_p_id



-- :name nntp_next
-- input: {board id} {old b_p_id}
SELECT
	xbp.b_p_id,
	xbp.g_p_id,
	xbp.msgid
FROM
	ib.bposts AS xbp
WHERE
	xbp.b_id = $1 AND xbp.b_p_id > $2
ORDER BY
	xbp.b_p_id ASC
LIMIT
	1

-- :name nntp_last
-- input: {board id} {old b_p_id}
SELECT
	xbp.b_p_id,
	xbp.g_p_id,
	xbp.msgid
FROM
	ib.bposts AS xbp
WHERE
	xbp.b_id = $1 AND xbp.b_p_id < $2
ORDER BY
	xbp.b_p_id DESC
LIMIT
	1



-- :name nntp_newnews_all
-- input: {time since}
SELECT DISTINCT ON (xbp.date_recv,xbp.g_p_id)
	xbp.msgid
FROM
	ib.bposts AS xbp
WHERE
	xbp.date_recv >= $1
ORDER BY
	xbp.date_recv,
	xbp.g_p_id

-- :name nntp_newnews_one
-- input: {time since} {board name}
SELECT
	xbp.msgid
FROM
	ib.boards AS xb
JOIN
	ib.bposts AS xbp
USING
	(b_id)
WHERE
	xbp.date_recv >= $1 AND
		xb.newsgroup = $2
ORDER BY
	xbp.date_recv,
	xbp.g_p_id

-- :name nntp_newnews_all_group
-- input: {time since}
-- clientside filtering of multiple posts to one board
SELECT
	xbp.msgid,
	xb.newsgroup
FROM
	ib.boards AS xb
JOIN
	ib.bposts AS xbp
USING
	(b_id)
WHERE
	xbp.date_recv >= $1
ORDER BY
	xbp.date_recv,
	xbp.g_p_id



-- :name nntp_newgroups
-- input: {time since}
SELECT
	xb.newsgroup,
	MIN(xbp.b_p_id) AS lo,
	MAX(xbp.b_p_id) AS hi
FROM
	ib.boards AS xb
LEFT JOIN
	ib.bposts AS xbp
USING
	(b_id)
WHERE
	xb.b_added >= $1
GROUP BY
	xb.b_id
ORDER BY
	xb.b_added,
	xb.b_id



-- :name nntp_listactive_all
SELECT
	xb.newsgroup,
	MIN(xbp.b_p_id),
	MAX(xbp.b_p_id)
FROM
	ib.boards AS xb
LEFT JOIN
	ib.bposts AS xbp
USING
	(b_id)
GROUP BY
	xb.b_id
ORDER BY
	xb.newsgroup COLLATE "und-x-icu"

-- :name nntp_listactive_one
-- input: {board name}
SELECT
	xb.newsgroup,
	MIN(xbp.b_p_id),
	MAX(xbp.b_p_id)
FROM
	ib.boards AS xb
LEFT JOIN
	ib.bposts AS xbp
USING
	(b_id)
WHERE
	xb.newsgroup = $1
GROUP BY
	xb.b_id



-- :name nntp_over_msgid
-- input: {msgid}
-- string_agg(xb.newsgroup || ':' || xbp.b_p_id, ' ') -- unused
-- TODO filter by xp.date_recv/xbp.date_recv
SELECT
	array_agg(xbp.b_id),
	array_agg(xbp.b_p_id),
	array_agg(xb.newsgroup),
	xp.title,
	xp.headers -> 'Subject' ->> 0,
	xp.headers -> 'From' ->> 0,
	xp.headers -> 'Date' ->> 0,
	xp.headers -> 'References' ->> 0,
	(xp.date_recv IS NULL)
FROM
	ib.boards AS xb
JOIN
	ib.bposts AS xbp
USING
	(b_id)
JOIN
	ib.gposts AS xp
USING
	(g_p_id)
WHERE
	xp.msgid = $1
GROUP BY
	xp.g_p_id

-- :name nntp_over_range
-- input: {bid} {min} {max}
SELECT
	array_agg(zbp.b_id),
	array_agg(zbp.b_p_id),
	array_agg(zb.newsgroup),
	xbp.b_p_id,
	xp.msgid,
	xp.title,
	xp.headers -> 'Subject' ->> 0,
	xp.headers -> 'From' ->> 0,
	xp.headers -> 'Date' ->> 0,
	xp.headers -> 'References' ->> 0
FROM
	ib.bposts AS xbp
JOIN
	ib.gposts AS xp
ON
	xbp.g_p_id = xp.g_p_id
JOIN
	ib.bposts AS zbp
ON
	xp.g_p_id = zbp.g_p_id
JOIN
	ib.boards AS zb
ON
	zbp.b_id = zb.b_id
WHERE
	xbp.b_id = $1 AND
		xbp.b_p_id >= $2 AND
		($3 < 0 OR xbp.b_p_id <= $3)
GROUP BY
	xp.g_p_id,
	xbp.b_p_id
ORDER BY
	xbp.b_p_id ASC

-- :name nntp_over_curr
-- input: {gpid}
SELECT
	array_agg(xbp.b_id),
	array_agg(xbp.b_p_id),
	array_agg(xb.newsgroup),
	xp.msgid,
	xp.title,
	xp.headers -> 'Subject' ->> 0,
	xp.headers -> 'From' ->> 0,
	xp.headers -> 'Date' ->> 0,
	xp.headers -> 'References' ->> 0
FROM
	ib.boards AS xb
JOIN
	ib.bposts AS xbp
USING
	(b_id)
JOIN
	ib.gposts AS xp
USING
	(g_p_id)
WHERE
	xp.g_p_id = $1
GROUP BY
	xp.g_p_id



-- :name nntp_hdr_msgid_msgid
-- input: {msgid} {bid}
SELECT
	xbp.b_id,
	xbp.b_p_id,
	(xp.date_recv IS NULL)
FROM
	ib.gposts AS xp
JOIN
	ib.bposts AS xbp
USING
	(g_p_id)
WHERE
	xp.msgid = $1
ORDER BY
	(xbp.b_id = $2) DESC
LIMIT
	1

-- :name nntp_hdr_msgid_subject
-- input: {msgid} {bid}
SELECT
	xbp.b_id,
	xbp.b_p_id,
	xp.title,
	xp.headers -> 'Subject' ->> 0,
	(xp.date_recv IS NULL)
FROM
	ib.gposts AS xp
JOIN
	ib.bposts AS xbp
USING
	(g_p_id)
WHERE
	xp.msgid = $1
ORDER BY
	(xbp.b_id = $2) DESC
LIMIT
	1

-- :name nntp_hdr_msgid_any
-- input: {msgid} {bid}
SELECT
	xbp.b_id,
	xbp.b_p_id,
	xp.headers -> $3 ->> 0,
	(xp.date_recv IS NULL)
FROM
	ib.gposts AS xp
JOIN
	ib.bposts AS xbp
USING
	(g_p_id)
WHERE
	xp.msgid = $1
ORDER BY
	(xbp.b_id = $2) DESC
LIMIT
	1


-- :name nntp_hdr_range_msgid
-- input: bid min max
SELECT
	xbp.b_p_id,
	'<' || xp.msgid || '>'
FROM
	ib.gposts AS xp
JOIN
	ib.bposts AS xbp
USING
	(g_p_id)
WHERE
	xbp.b_id = $1 AND
		xbp.b_p_id >= $2 AND ($3 < 0 OR xbp.b_p_id <= $3)
ORDER BY
	xbp.b_p_id ASC

-- :name nntp_hdr_range_subject
-- input: bid min max
SELECT
	xbp.b_p_id,
	xp.title,
	xp.headers -> 'Subject' ->> 0
FROM
	ib.gposts AS xp
JOIN
	ib.bposts AS xbp
USING
	(g_p_id)
WHERE
	xbp.b_id = $1 AND
		xbp.b_p_id >= $2 AND ($3 < 0 OR xbp.b_p_id <= $3)
ORDER BY
	xbp.b_p_id ASC

-- :name nntp_hdr_range_any
-- input: bid min max hdr
SELECT
	xbp.b_p_id,
	xp.headers -> $4 ->> 0
FROM
	ib.gposts AS xp
JOIN
	ib.bposts AS xbp
USING
	(g_p_id)
WHERE
	xbp.b_id = $1 AND
		xbp.b_p_id >= $2 AND ($3 < 0 OR xbp.b_p_id <= $3)
ORDER BY
	xbp.b_p_id ASC


-- :name nntp_hdr_curr_msgid
-- input: gpid
SELECT
	'<' || msgid || '>'
FROM
	ib.gposts
WHERE
	g_p_id = $1
LIMIT
	1

-- :name nntp_hdr_curr_subject
-- input: gpid
SELECT
	title,
	headers -> 'Subject' ->> 0
FROM
	ib.gposts
WHERE
	g_p_id = $1
LIMIT
	1

-- :name nntp_hdr_curr_any
-- input: gpid hdr
SELECT
	headers -> $2 ->> 0
FROM
	ib.gposts
WHERE
	g_p_id = $1
LIMIT
	1
