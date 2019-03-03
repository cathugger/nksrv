-- :name web_listboards
SELECT
	b_name,bdesc,attrib,t_count,p_count
FROM
	ib0.boards
ORDER BY
	b_name

-- :name web_thread_list_page
-- input: {b_name} {page num}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xb.threads_per_page,
	xb.t_count,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xbp.b_p_id,
	xbp.p_name,
	xp.msgid,
	xp.pdate,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xp.attrib,
	xp.headers,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.fsize,
	xf.thumb,
	xf.oname,
	xf.filecfg,
	xf.thumbcfg
FROM
	(
		SELECT
			b_id,
			bdesc,
			attrib,
			threads_per_page,
			t_count
		FROM
			ib0.boards
		WHERE
			b_name=$1
		LIMIT
			1
	) AS xb
LEFT JOIN
	LATERAL (
		SELECT
			zt.b_id,
			zt.t_id,
			zt.t_name,
			zt.bump,
			zt.p_count,
			zt.f_count
		FROM
			ib0.threads AS zt
		WHERE
			zt.b_id = xb.b_id
		ORDER BY
			zt.bump DESC,
			zt.t_id ASC
		LIMIT
			(CASE
				WHEN
					xb.threads_per_page > 0
				THEN
					xb.threads_per_page
				WHEN
					$2 = 0
				THEN
					NULL
				ELSE
					0
				END
			)
		OFFSET
			(CASE
				WHEN
					xb.threads_per_page > 0
				THEN
					$2 * xb.threads_per_page
				ELSE
					0
				END
			)
	) AS xt
ON
	TRUE
LEFT JOIN
	LATERAL (
		SELECT
			b_id,
			t_id,
			b_p_id,
			g_p_id,
			p_name,
			pdate
		FROM
			ib0.bposts AS op
		WHERE
			op.b_id = xb.b_id AND op.b_p_id = xt.t_id
		UNION ALL
		SELECT
			b_id,
			t_id,
			b_p_id,
			g_p_id,
			p_name,
			pdate
		FROM (
			SELECT *
			FROM
				(
					SELECT
						rp.b_id,
						rp.t_id,
						rp.b_p_id,
						rp.g_p_id,
						rp.p_name,
						rp.pdate
					FROM
						ib0.bposts AS rp
					WHERE
						rp.b_id = xb.b_id AND
							rp.t_id = xt.t_id AND
							rp.b_p_id != xt.t_id
					ORDER BY
						rp.pdate DESC,
						rp.b_p_id DESC
					LIMIT 5
				) AS tt
			ORDER BY
				pdate ASC,
				b_p_id ASC
		) AS ttt
	) AS xbp
ON
	TRUE
LEFT JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN
	ib0.files AS xf
ON
	xp.g_p_id = xf.g_p_id
ORDER BY
	xt.bump DESC,
	xt.t_id ASC,
	xbp.pdate ASC,
	xbp.b_p_id ASC,
	xf.f_id ASC

-- :name web_overboard_page
-- input: {page num} {threads_per_page}
SELECT
	xb.b_id,
	xb.b_name,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xbp.b_p_id,
	xbp.p_name,
	xp.msgid,
	xp.pdate,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xp.attrib,
	xp.headers,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.fsize,
	xf.thumb,
	xf.oname,
	xf.filecfg,
	xf.thumbcfg
FROM
	(
		SELECT
			zt.b_id,
			zt.t_id,
			zt.t_name,
			zt.bump,
			zt.p_count,
			zt.f_count
		FROM
			ib0.threads AS zt
		WHERE
			zt.skip_over IS NOT TRUE
		ORDER BY
			zt.bump DESC,
			zt.b_id ASC,
			zt.t_id ASC
		LIMIT
			(CASE
				WHEN
					$2 > 0
				THEN
					$2
				WHEN
					$1 = 0
				THEN
					NULL
				ELSE
					0
				END
			)
		OFFSET
			(CASE
				WHEN
					$2 > 0
				THEN
					$1 * $2
				ELSE
					0
				END
			)
	) AS xt
JOIN
	ib0.boards AS xb
ON
	xt.b_id = xb.b_id
LEFT JOIN
	LATERAL (
		SELECT
			b_id,
			t_id,
			b_p_id,
			g_p_id,
			p_name,
			pdate
		FROM
			ib0.bposts AS op
		WHERE
			op.b_id = xt.b_id AND op.b_p_id = xt.t_id
		UNION ALL
		SELECT
			b_id,
			t_id,
			b_p_id,
			g_p_id,
			p_name,
			pdate
		FROM (
			SELECT *
			FROM
				(
					SELECT
						rp.b_id,
						rp.t_id,
						rp.b_p_id,
						rp.g_p_id,
						rp.p_name,
						rp.pdate
					FROM
						ib0.bposts AS rp
					WHERE
						rp.b_id = xt.b_id AND
							rp.t_id = xt.t_id AND
							rp.b_p_id != xt.t_id
					ORDER BY
						rp.pdate DESC,
						rp.b_p_id DESC
					LIMIT 5
				) AS tt
			ORDER BY
				pdate ASC,
				b_p_id ASC
		) AS ttt
	) AS xbp
ON
	TRUE
LEFT JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN
	ib0.files AS xf
ON
	xp.g_p_id = xf.g_p_id
ORDER BY
	xt.bump DESC,
	xt.t_id ASC,
	xbp.pdate ASC,
	xbp.b_p_id ASC,
	xf.f_id ASC

-- :name web_thread_catalog
-- input: {b_name}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xt.bump,
	xbp.b_p_id,
	xp.pdate,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.thumb,
	xf.thumbcfg
FROM
	ib0.boards AS xb
LEFT JOIN
	ib0.threads AS xt
ON
	xb.b_id = xt.b_id
LEFT JOIN
	ib0.bposts AS xbp
ON
	xt.b_id = xbp.b_id AND xt.t_id = xbp.b_p_id
LEFT JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN
	LATERAL (
		SELECT
			zf.f_id,
			zf.fname,
			zf.ftype,
			zf.thumb,
			zf.thumbcfg
		FROM
			ib0.files AS zf
		WHERE
			xp.g_p_id = zf.g_p_id AND zf.ftype != 'msg'
		ORDER BY
			zf.f_id
		LIMIT
			1
	) AS xf
ON
	TRUE
WHERE
	xb.b_name = $1
ORDER BY
	xt.bump DESC,
	xt.t_id ASC,
	xf.f_id ASC

-- :name web_thread
-- input: {b_name} {t_name}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xb.threads_per_page,
	xb.t_count,
	xt.t_id,
	xt.t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xto.t_pos,
	xbp.b_p_id,
	xbp.p_name,
	xp.msgid,
	xp.pdate,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
	xp.attrib,
	xp.headers,
	xf.f_id,
	xf.fname,
	xf.ftype,
	xf.fsize,
	xf.thumb,
	xf.oname,
	xf.filecfg,
	xf.thumbcfg
FROM
	ib0.boards AS xb
LEFT JOIN
	ib0.threads AS xt
ON
	xb.b_id = xt.b_id
LEFT JOIN
	LATERAL (
		SELECT
			*
		FROM
			(
				SELECT
					b_id,
					t_id,
					row_number() OVER (
						ORDER BY
							bump DESC,
							t_id ASC
					) AS t_pos
				FROM
					ib0.threads qt
				WHERE
					qt.b_id = xt.b_id
			) AS zt
		WHERE
			xt.b_id = zt.b_id AND xt.t_id = zt.t_id
		LIMIT
			1
	) AS xto
ON
	TRUE
LEFT JOIN
	ib0.bposts AS xbp
ON
	xt.b_id = xbp.b_id AND xt.t_id = xbp.t_id
LEFT JOIN
	ib0.posts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN
	ib0.files AS xf
ON
	xp.g_p_id = xf.g_p_id
WHERE
	xb.b_name=$1 AND xt.t_name = $2
ORDER BY
	xbp.pdate ASC,
	xbp.b_p_id ASC,
	xf.f_id ASC



-- TODO common bucket

-- :name web_failref_write
WITH
	delold AS (
		DELETE FROM
			ib0.failrefs
		WHERE
			g_p_id = $1
	)
INSERT INTO
	ib0.failrefs (
		g_p_id,
		p_name,
		b_name,
		msgid
	)
SELECT
	$1,
	unnest($2::text[]) AS p_name,
	unnest($3::text[]) AS b_name,
	unnest($4::text[]) AS msgid

-- :name web_failref_find
-- args: p_name,board,msgid
WITH
	msgs AS (
		SELECT
			g_p_id
		FROM
			ib0.failrefs
		WHERE
			(p_name LIKE substring($1 for 8) || '%') AND
				($1 LIKE p_name || '%') AND
				(b_name IS NULL OR b_name = $2)
		UNION
		SELECT
			g_p_id
		FROM
			ib0.failrefs
		WHERE
			msgid = $3
		LIMIT
			8192
	)
SELECT
	msgs.g_p_id,
	xp.message,
	xp.headers -> 'In-Reply-To' ->> 0,
	xp.attrib,
	xbp.b_id,
	xbp.t_id
FROM
	msgs
JOIN
	ib0.posts AS xp
ON
	xp.g_p_id = msgs.g_p_id
JOIN
	LATERAL (
		SELECT
			zbp.b_id,
			zbp.t_id
		FROM
			ib0.bposts AS zbp
		JOIN
			ib0.boards AS zb
		ON
			zbp.b_id = zb.b_id
		WHERE
			zbp.g_p_id = xp.g_p_id
		ORDER BY
			zb.b_name
		LIMIT
			1
	) AS xbp
ON
	TRUE

-- :name update_post_attrs
UPDATE
	ib0.posts
SET
	attrib = $2
WHERE
	g_p_id = $1


-- :name autoregister_mod
INSERT INTO
	ib0.modlist AS ml (
		mod_pubkey,
		automanage
	)
VALUES (
	$1,
	TRUE
)
ON CONFLICT (mod_pubkey) DO UPDATE -- DO NOTHING returns nothing so we update something irrelevant as hack
	SET automanage = ml.automanage
RETURNING
	mod_id, mod_priv

