-- :name web_listboards
SELECT
	b_id,
	b_name,
	bdesc,
	attrib,
	t_count,
	p_count
FROM
	ib0.boards
ORDER BY
	b_name COLLATE "und-x-icu"

-- :name web_thread_list_page
-- input: {b_name} {page num}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xb.threads_per_page,
	xb.t_count,
	xt.b_t_id,
	xt.b_t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xbp.b_p_id,
	xbp.p_name,
	xbp.activ_refs,
	xp.msgid,
	xp.date_sent,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
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
	ib0.boards xb
LEFT JOIN
	LATERAL (
		SELECT
			zt.b_id,
			zt.b_t_id,
			zt.b_t_name,
			zt.bump,
			zt.p_count,
			zt.f_count
		FROM
			ib0.threads AS zt
		WHERE
			zt.b_id = xb.b_id
		ORDER BY
			zt.bump DESC,
			zt.b_t_id ASC
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
			b_t_id,
			b_p_id,
			g_p_id,
			p_name,
			date_sent,
			activ_refs
		FROM
			ib0.bposts AS op
		WHERE
			op.b_id = xb.b_id AND op.b_p_id = xt.b_t_id

		UNION ALL

		SELECT
			b_id,
			b_t_id,
			b_p_id,
			g_p_id,
			p_name,
			date_sent,
			activ_refs
		FROM (
			SELECT
				*
			FROM
				(
					SELECT
						rp.b_id,
						rp.b_t_id,
						rp.b_p_id,
						rp.g_p_id,
						rp.p_name,
						rp.date_sent,
						rp.activ_refs
					FROM
						ib0.bposts AS rp
					WHERE
						rp.b_id = xb.b_id AND
							rp.b_t_id = xt.b_t_id AND
							rp.b_p_id != xt.b_t_id
					ORDER BY
						rp.date_sent DESC,
						rp.b_p_id    DESC
					LIMIT 5
				) AS tt
			ORDER BY
				date_sent ASC,
				b_p_id    ASC
		) AS ttt
	) AS xbp
ON
	TRUE
LEFT JOIN
	ib0.gposts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.files zf
		WHERE
			xp.g_p_id = zf.g_p_id
		ORDER BY
			zf.f_id
	) AS xf
ON
	TRUE
WHERE
	xb.b_name = $1

-- :name web_overboard_page
-- input: {page num} {threads_per_page}
SELECT
	xt.b_id,
	xt.b_name,
	xt.b_t_id,
	xt.b_t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xbp.b_p_id,
	xbp.p_name,
	xbp.activ_refs,
	xp.msgid,
	xp.date_sent,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
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
			zb.b_name,
			zt.b_t_id,
			zt.b_t_name,
			zt.bump,
			zt.p_count,
			zt.f_count
		FROM
			ib0.threads AS zt
		JOIN
			ib0.boards AS zb
		ON
			zt.b_id = zb.b_id
		WHERE
			zt.skip_over IS NOT TRUE
		ORDER BY
			zt.bump DESC,
			zt.g_t_id ASC,
			zt.b_id ASC
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
LEFT JOIN
	LATERAL (
		SELECT
			b_id,
			b_t_id,
			b_p_id,
			g_p_id,
			p_name,
			date_sent,
			activ_refs
		FROM
			ib0.bposts AS op
		WHERE
			op.b_id = xt.b_id AND op.b_p_id = xt.b_t_id

		UNION ALL

		SELECT
			b_id,
			b_t_id,
			b_p_id,
			g_p_id,
			p_name,
			date_sent,
			activ_refs
		FROM (
			SELECT
				*
			FROM
				(
					SELECT
						rp.b_id,
						rp.b_t_id,
						rp.b_p_id,
						rp.g_p_id,
						rp.p_name,
						rp.date_sent,
						rp.activ_refs
					FROM
						ib0.bposts AS rp
					WHERE
						rp.b_id = xt.b_id AND
							rp.b_t_id = xt.b_t_id AND
							rp.b_p_id != xt.b_t_id
					ORDER BY
						rp.date_sent DESC,
						rp.b_p_id    DESC
					LIMIT 5
				) AS tt
			ORDER BY
				date_sent ASC,
				b_p_id    ASC
		) AS ttt
	) AS xbp
ON
	TRUE
-- XXX possibly misorder join, too annoy to move inside
LEFT JOIN
	ib0.gposts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.files zf
		WHERE
			xp.g_p_id = zf.g_p_id
		ORDER BY
			zf.f_id ASC
	) AS xf
ON
	TRUE

-- :name web_thread_catalog
-- input: {b_name}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xt.b_t_id,
	xt.b_t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xt.bump,
	xbp.b_p_id,
	xp.date_sent,
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
	ib0.boards xb
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.threads zt
		WHERE
			xb.b_id = zt.b_id
		ORDER BY
			zt.bump DESC,
			zt.b_t_id ASC
	) AS xt
ON
	TRUE
LEFT JOIN
	ib0.bposts xbp
ON
	xt.b_id = xbp.b_id AND xt.b_t_id = xbp.b_p_id
LEFT JOIN
	ib0.gposts xp
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

-- :name web_overboard_catalog
-- input: {thread_count}
SELECT
	xt.b_id,
	xt.b_name,
	xt.b_t_id,
	xt.b_t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xt.bump,
	xbp.b_p_id,
	xp.date_sent,
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
	(
		SELECT
			zt.b_id,
			zb.b_name,
			zt.b_t_id,
			zt.b_t_name,
			zt.bump,
			zt.p_count,
			zt.f_count
		FROM
			ib0.threads AS zt
		JOIN
			ib0.boards AS zb
		ON
			zt.b_id = zb.b_id
		WHERE
			zt.skip_over IS NOT TRUE
		ORDER BY
			zt.bump DESC,
			zt.g_t_id ASC,
			zt.b_id ASC
		LIMIT
			$1
	) AS xt
LEFT JOIN
	ib0.bposts xbp
ON
	xt.b_id = xbp.b_id AND xt.b_t_id = xbp.b_p_id
LEFT JOIN
	ib0.gposts xp
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

-- :name web_thread
-- input: {b_name} {t_name}
SELECT
	xb.b_id,
	xb.bdesc,
	xb.attrib,
	xb.threads_per_page,
	xb.t_count,
	xt.b_t_id,
	xt.b_t_name,
	xt.p_count,
	xt.f_count AS xt_f_count,
	xto.t_pos,
	xbp.b_p_id,
	xbp.p_name,
	xbp.activ_refs,
	xp.msgid,
	xp.date_sent,
	xp.sage,
	xp.f_count AS xp_f_count,
	xp.author,
	xp.trip,
	xp.title,
	xp.message,
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
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.threads zt
		WHERE
			zt.b_id = xb.b_id AND zt.b_t_name = $2
		LIMIT
			1
	) AS xt
ON
	TRUE
LEFT JOIN
	LATERAL (
		SELECT
			*
		FROM
			(
				SELECT
					b_id,
					b_t_id,
					row_number() OVER (
						ORDER BY
							bump DESC,
							b_t_id ASC
					) AS t_pos
				FROM
					ib0.threads qt
				WHERE
					qt.b_id = xt.b_id
			) AS zt
		WHERE
			xt.b_id = zt.b_id AND xt.b_t_id = zt.b_t_id
		LIMIT
			1
	) AS xto
ON
	TRUE
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.bposts zbp
		WHERE
			xt.b_id = zbp.b_id AND xt.b_t_id = zbp.b_t_id
		ORDER BY
			zbp.date_sent ASC,
			zbp.b_p_id    ASC
	) AS xbp
ON
	TRUE
LEFT JOIN
	ib0.gposts AS xp
ON
	xbp.g_p_id = xp.g_p_id
LEFT JOIN LATERAL
	(
		SELECT
			*
		FROM
			ib0.files zf
		WHERE
			xp.g_p_id = zf.g_p_id
		ORDER BY
			zf.f_id ASC
	) AS xf
ON
	TRUE
WHERE
	xb.b_name=$1


-- :name web_prepost_newthread
SELECT
	b_id,
	post_limits,
	newthread_limits
FROM
	ib0.boards
WHERE
	b_name=$1

-- :name web_prepost_newpost
WITH
	xb AS (
		SELECT
			b_id,
			post_limits,
			reply_limits,
			thread_opts
		FROM
			ib0.boards
		WHERE
			b_name=$1
		LIMIT
			1
	)
SELECT
	xb.b_id,
	xb.post_limits,
	xb.reply_limits,
	xtp.b_t_id,
	xtp.reply_limits,
	xb.thread_opts,
	xtp.thread_opts,
	xtp.msgid,
	xtp.date_sent
FROM
	xb
LEFT JOIN LATERAL
	(
		SELECT
			xt.b_id,
			xt.b_t_id,
			xt.reply_limits,
			xt.thread_opts,
			xp.msgid,
			xp.date_sent
		FROM
			ib0.threads xt
		JOIN
			ib0.bposts xbp
		ON
			xt.b_id=xbp.b_id AND xt.b_t_id=xbp.b_p_id
		JOIN
			ib0.gposts xp
		ON
			xbp.g_p_id = xp.g_p_id
		WHERE
			xb.b_id = xt.b_id AND xt.b_t_name=$2
		LIMIT
			1
	) AS xtp
ON
	TRUE
