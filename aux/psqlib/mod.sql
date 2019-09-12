-- modification related queries

-- :name mod_ref_write
-- NOTE: this used to track failed references, therefore needed update
-- it used to contain delete stmt, but since we now track all refs,
-- it's no longer needed
INSERT INTO
	ib0.refs (
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
					-- index-search by first 8 chars, then narrow
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
	xbp.b_t_id
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

-- :name mod_autoregister_mod
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



-- :namet mod_delete_msgid_common
	delbp AS (
		-- delete all board posts of that
		DELETE FROM
			ib0.bposts xbp
		USING
			delgp
		WHERE
			xbp.g_p_id = delgp.g_p_id
		RETURNING
			xbp.b_id,
			xbp.b_t_id,
			xbp.b_p_id,
			xbp.p_name,
			xbp.msgid,
			xbp.mod_id,
			delgp.f_count
	),
	delbt AS (
		-- delete thread(s) incase we nuked OP(s)
		DELETE FROM
			ib0.threads xt
		USING
			delbp
		WHERE
			xt.b_id = delbp.b_id AND
				xt.b_t_id = delbp.b_p_id
		RETURNING
			xt.b_id,
			xt.b_t_id
	),
	updbt AS (
		-- update thread(s) counters incase we haven't deleted thread(s) earlier
		-- un-bump is done adhoc
		UPDATE
			ib0.threads xt
		SET
			p_count = xt.p_count - 1,
			f_count = xt.f_count - delbp.f_count
		FROM
			delbp
		WHERE
			delbp.b_id = xt.b_id AND
				delbp.b_t_id = xt.b_t_id
	),
	delbcp AS (
		-- delete board child posts incase we nuked thread(s)
		DELETE FROM
			ib0.bposts xbp
		USING
			delbt
		WHERE
			xbp.b_id = delbt.b_id AND
				xbp.b_t_id = delbt.b_t_id
		RETURNING
			xbp.b_id,
			xbp.b_p_id,
			xbp.p_name,
			xbp.msgid,
			xbp.g_p_id,
			xbp.mod_id
	),
	delgcp AS (
		-- delete global child posts (from above)
		-- (if they dont have refs from other boards)
		-- XXX but how children of thread could have extra refs???
		DELETE FROM
			ib0.posts xp
		USING
			(
				-- XXX is it even possible to have this false?
				SELECT
					delbcp.g_p_id,COUNT(xbp.g_p_id) > 1 AS hasrefs
				FROM
					delbcp
				LEFT JOIN
					ib0.bposts xbp
				ON
					delbcp.g_p_id = xbp.g_p_id
				GROUP BY
					delbcp.g_p_id
			) AS rcnts
		WHERE
			rcnts.hasrefs = FALSE AND
				rcnts.g_p_id = xp.g_p_id
		RETURNING
			xp.g_p_id,
			xp.msgid
	),
	clean_mods AS (
		-- garbage collect moderator list (maybe we nuked mod post(s))
		DELETE FROM
			ib0.modlist mods
		USING
			(
				SELECT
					delmod.mod_id,COUNT(xbp.mod_id) > 1 AS hasrefs
				FROM
					(
						SELECT mod_id,b_id,b_p_id FROM delbp
						UNION ALL
						SELECT mod_id,b_id,b_p_id FROM delbcp
					) AS delmod
				LEFT JOIN
					ib0.bposts xbp
				ON
					delmod.mod_id = xbp.mod_id
				WHERE
					delmod.mod_id IS NOT NULL
				GROUP BY
					delmod.mod_id
			) AS rcnts
		WHERE
			rcnts.hasrefs = FALSE AND
				rcnts.mod_id = mods.mod_id AND
				mods.automanage = TRUE
	),
	updb AS (
		-- update boards post and thread counts
		UPDATE
			ib0.boards xb
		SET
			p_count = xb.p_count - xtp.p_count,
			t_count = xb.t_count - xtp.t_count
		FROM
			(
				SELECT
					xx.b_id,
					SUM(xx.p_count) AS p_count,
					COUNT(delbt.b_id) AS t_count
				FROM
					(
						SELECT
							delbpx.b_id,
							COUNT(delbpx.b_id) AS p_count
						FROM
							(
								SELECT b_id FROM delbp
								UNION ALL
								SELECT b_id FROM delbcp
							) AS delbpx
						GROUP BY
							delbpx.b_id
					) AS xx
				LEFT JOIN
					delbt
				ON
					xx.b_id = delbt.b_id
				GROUP BY
					xx.b_id
			) AS xtp
		WHERE
			xb.b_id = xtp.b_id
	),
	delf AS (
		-- delete relevant files
		DELETE FROM
			ib0.files xf
		USING
			(
				SELECT g_p_id FROM delgp
				UNION ALL
				SELECT g_p_id FROM delgcp
			) AS xgpids
		WHERE
			xgpids.g_p_id = xf.g_p_id
		RETURNING
			xf.f_id,
			xf.fname,
			xf.thumb
	)

SELECT
	leftf.fname,leftf.fnum,leftt.thumb,leftt.tnum,
	NULL,NULL,NULL,NULL,NULL,NULL,NULL
FROM
	(
		-- minus 1 because snapshot isolation
		SELECT
			delf.fname,COUNT(xf.fname) - 1 AS fnum
		FROM
			delf
		LEFT JOIN
			ib0.files xf
		ON
			delf.fname = xf.fname
		GROUP BY
			delf.fname
	) AS leftf
FULL JOIN
	(
		-- minus 1 because snapshot isolation
		SELECT
			delf.fname,delf.thumb,COUNT(xf.thumb) - 1 AS tnum
		FROM
			delf
		LEFT JOIN
			ib0.files xf
		ON
			delf.fname = xf.fname AND
				delf.thumb = xf.thumb
		GROUP BY
			delf.fname,
			delf.thumb
	) AS leftt
ON
	leftf.fname = leftt.fname

UNION ALL

SELECT
	'',0,'',0,xt.b_id,xt.b_t_id,xto.t_pos,NULL,NULL,NULL,NULL
FROM
	delbp AS xt
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
WHERE
	xt.b_t_id != xt.b_p_id

UNION ALL

SELECT
	'',0,'',0,NULL,NULL,NULL,msgid,NULL,NULL,NULL
FROM
	delgp

UNION ALL

SELECT
	'',0,'',0,NULL,NULL,NULL,msgid,NULL,NULL,NULL
FROM
	delgcp

UNION ALL

SELECT
	'',0,'',0,NULL,NULL,NULL,NULL,xb.b_name,delbpx.p_name,delbpx.msgid
FROM
	(
		SELECT b_id,p_name,msgid FROM delbp
		UNION ALL
		SELECT b_id,p_name,msgid FROM delbcp
	) AS delbpx
JOIN
	ib0.boards xb
ON
	delbpx.b_id = xb.b_id


-- :name mod_delete_by_msgid
/*
IMPORTANT:
https://www.postgresql.org/docs/9.6/queries-with.html
All the statements are executed with the same snapshot (see Chapter 13),
so they cannot "see" one another's effects on the target tables.
This alleviates the effects of the unpredictability of the actual order
of row updates, and means that RETURNING data is the only way to
communicate changes between different WITH sub-statements and the main query.
*/
WITH
	delgp AS (
		-- delete global post
		DELETE FROM
			ib0.posts
		WHERE
			msgid = $1 AND
				padded IS NOT NULL
		RETURNING
			g_p_id,
			f_count,
			msgid
	),
	{{- .mod_delete_msgid_common }}

-- :name mod_ban_by_msgid
WITH
	insban AS (
		INSERT INTO
			ib0.banlist (
				msgid,
				b_id,
				b_p_id,
				ban_info
			)
		VALUES
			(
				$1,
				$2,
				$3,
				$4
			)
	),
	delgp AS (
		-- replace global post with ban post
		INSERT INTO
			ib0.posts xp (
				msgid,
				pdate,
				padded,
				sage,
				f_count,
				author,
				trip,
				title,
				message,
				headers,
				attrib,
				layout,
				extras
			)
		VALUES
			(
				$1,
				NULL,
				NULL,
				FALSE,
				0,
				'',
				'',
				'',
				'',
				NULL,
				NULL,
				NULL,
				NULL
			)
		ON CONFLICT (msgid) DO UPDATE
			SET
				pdate   = excluded.pdate,
				padded  = excluded.padded,
				sage    = excluded.sage,
				f_count = excluded.f_count,
				author  = excluded.author,
				trip    = excluded.trip,
				title   = excluded.title,
				message = excluded.message,
				headers = excluded.headers,
				attrib  = excluded.attrib,
				layout  = excluded.layout,
				extras  = excluded.extras
			WHERE
				xp.padded IS NOT NULL
		RETURNING
			g_p_id,
			f_count,
			msgid
	),
	{{- .mod_delete_msgid_common }}

-- :name mod_bname_topts_by_tid
-- returns boardname and thread opts
SELECT
	xb.b_name,xb.thread_opts,xt.thread_opts
FROM
	ib0.boards xb
JOIN
	ib0.threads xt
ON
	xb.b_id = xt.b_id
WHERE
	xb.b_id = $1 AND xt.b_t_id = $2

-- :name mod_refresh_bump_by_tid
UPDATE
	ib0.threads
SET
	bump = pdate
FROM
	(
		SELECT
			pdate
		FROM
			(
				SELECT
					pdate,
					b_p_id,
					sage
				FROM
					ib0.bposts
				WHERE
					-- count sages against bump limit.
					-- because others do it like that :<
					b_id = $1 AND b_t_id = $2
				ORDER BY
					pdate ASC,
					b_p_id ASC
				LIMIT
					$3
				-- take bump posts, sorted by original date,
				-- only upto bump limit
			) AS tt
		WHERE
			sage != TRUE
		ORDER BY
			pdate DESC,b_p_id DESC
		LIMIT
			1
		-- and pick latest one
	) as xbump
WHERE
	b_id = $1 AND b_t_id = $2


-- :name mod_set_mod_priv
-- args: <pubkey> <newpriv>
INSERT INTO
	ib0.modlist AS ml (
		mod_pubkey,
		automanage,
		mod_priv
	)
VALUES
	(
		$1,
		FALSE,
		$2
	)
ON CONFLICT (mod_pubkey) DO UPDATE
	SET
		automanage = FALSE,
		mod_priv = $2
	WHERE
		ml.mod_priv <> $2 OR ml.automanage <> FALSE
RETURNING -- inserted or modified
	mod_id

-- :name mod_unset_mod
-- args: <pubkey>
WITH
	-- do update there
	upd_mod AS (
		UPDATE
			ib0.modlist
		SET
			mod_priv = 'none', -- don't see point having anything else there yet
			automanage = TRUE
		WHERE
			mod_pubkey = $1 AND
			(mod_priv <> 'none' OR automanage <> TRUE)
		RETURNING
			mod_id
	)
-- garbage collect moderator list
DELETE FROM
	ib0.modlist mods
USING
	(
		SELECT
			delmod.mod_id,COUNT(xbp.mod_id) > 0 AS hasrefs
		FROM
			upd_mod AS delmod
		LEFT JOIN
			ib0.bposts xbp
		ON
			delmod.mod_id = xbp.mod_id
		GROUP BY
			delmod.mod_id
	) AS rcnts
WHERE
	rcnts.hasrefs = FALSE AND rcnts.mod_id = mods.mod_id

-- :name mod_fetch_and_clear_mod_msgs
-- args: <modid> <offset>
-- fetches all messages of mod, and also clears all their actions
WITH
	zbp AS (
		SELECT
			b_id,
			b_p_id,
			b_t_id,
			g_p_id
		FROM
			ib0.bposts
		WHERE
			mod_id = $1
		ORDER BY
			b_id,b_p_id
		OFFSET
			$2
		LIMIT
			4096
	),
	zd AS (
		DELETE FROM
			ib0.banlist bl
		USING
			zbp
		WHERE
			bl.b_id = zbp.b_id AND bl.b_p_id = zbp.b_p_id
	)
SELECT
	zbp.g_p_id,
	zbp.b_id,
	zbp.b_p_id,
	yb.b_name,
	yp.msgid,
	ypp.msgid,
	yp.title,
	yp.pdate,
	yp.message,
	yp.extras -> 'text_attach',
	yf.fname
FROM
	zbp
-- board
JOIN
	ib0.boards yb
ON
	zbp.b_id = yb.b_id
-- global post
JOIN
	ib0.posts yp
ON
	zbp.g_p_id = yp.g_p_id
-- files of global post
LEFT JOIN LATERAL
	(
		SELECT
			xf.fname
		FROM
			ib0.files xf
		WHERE
			yp.g_p_id = xf.g_p_id
		ORDER BY
			xf.f_id -- important
	) AS yf
ON
	TRUE
-- parent board post
LEFT JOIN
	ib0.bposts ypbp
ON
	zbp.b_id = ypbp.b_id AND zbp.b_t_id = ypbp.b_p_id AND zbp.b_t_id != zbp.b_p_id
-- parent global post
LEFT JOIN
	ib0.posts ypp
ON
	ypbp.g_p_id = ypp.g_p_id
