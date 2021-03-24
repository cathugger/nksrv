-- modification related queries

-- :name mod_ref_write
-- NOTE: this used to track failed references, therefore needed update
-- it used to contain delete stmt, but since we now track all refs,
-- it's no longer needed
INSERT INTO
	ib.refs (
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
					ib.refs
				WHERE
					-- index-search by first 8 chars, then narrow
					(p_name LIKE substring($3 for 8) || '%') AND
						($3 LIKE p_name || '%') AND
						(b_name IS NULL OR b_name = $4)

				UNION

				SELECT
					b_id,b_p_id
				FROM
					ib.refs
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
	xbp.activ_refs,
	xbp.b_t_id
FROM
	msgs
JOIN
	ib.bposts AS xbp
ON
	xbp.b_id = msgs.b_id AND xbp.b_p_id = msgs.b_p_id
JOIN
	ib.gposts AS xp
ON
	xp.g_p_id = xbp.g_p_id

-- :name mod_update_bpost_activ_refs
UPDATE
	ib.bposts
SET
	activ_refs = $3
WHERE
	(b_id,b_p_id) = ($1,$2)

-- :name mod_autoregister_mod
INSERT INTO
	ib.modlist AS ml (
		mod_pubkey,
		automanage
	)
VALUES (
	$1,
	TRUE
)
ON CONFLICT (mod_pubkey) DO UPDATE -- DO NOTHING returns nothing so we update something irrelevant as hack
SET
	automanage = ml.automanage
RETURNING
	mod_id,

	mod_cap,
	mod_bcap,
	mod_caplvl,
	mod_bcaplvl,

	modi_cap,
	modi_bcap,
	modi_caplvl,
	modi_bcaplvl

-- :name mod_delete_by_msgid
-- lazyness: delete and then reinsert, this will fire foreign key cascades
-- XXX could probably just update and do proper triggers
WITH
	dgp AS (
		DELETE FROM
			ib.gposts
		WHERE
			msgid = $1 AND date_recv IS NOT NULL
		RETURNING
			msgid,
			has_ph,
			ph_ban,
			ph_banpriv
	)
INSERT INTO
	ib.gposts
	(
		msgid,
		has_ph,
		ph_ban,
		ph_banpriv
	)
SELECT
	msgid,
	has_ph,
	ph_ban,
	ph_banpriv
FROM
	dgp
WHERE
	has_ph IS TRUE


-- :name mod_ban_by_msgid
-- trigger will do its job
INSERT INTO
	ib.banlist
	(
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

-- :name mod_bname_topts_by_tid
-- returns boardname and thread opts
SELECT
	xb.b_name,
	xb.thread_opts,
	xt.thread_opts
FROM
	ib.boards xb
JOIN
	ib.threads xt
ON
	xb.b_id = xt.b_id
WHERE
	xb.b_id = $1 AND xt.b_t_id = $2

-- :name mod_refresh_bump_by_tid
UPDATE
	ib.threads
SET
	bump = date_sent
FROM
	(
		SELECT
			date_sent
		FROM
			(
				SELECT
					date_sent,
					b_p_id,
					sage
				FROM
					ib.bposts
				WHERE
					-- count sages against bump limit.
					-- because others do it like that :<
					b_id = $1 AND b_t_id = $2
				ORDER BY
					date_sent ASC,
					b_p_id    ASC
				LIMIT
					$3
				-- take bump posts, sorted by original date,
				-- only upto bump limit
			) AS tt
		WHERE
			sage != TRUE
		ORDER BY
			date_sent DESC,
			b_p_id    DESC
		LIMIT
			1
		-- and pick latest one
	) as xbump
WHERE
	b_id = $1 AND b_t_id = $2


-- :name mod_set_mod_priv
-- args: <pubkey> <capabilities> <delpriv>
INSERT INTO
	ib.modsets AS ms
	(
		mod_pubkey,

		mod_cap,
		mod_caplvl,
		modi_cap,
		modi_caplvl
	)
VALUES
	(
		$1,

		$2,
		$3,
		$4,
		$5
	)
ON CONFLICT (mod_pubkey)
WHERE
	b_id IS NULL AND b_p_id IS NULL AND
		mod_group IS NULL
DO UPDATE
	SET
		mod_cap     = $2,
		mod_caplvl  = $3,
		modi_cap    = $4,
		modi_caplvl = $5
	WHERE
		(
			ms.mod_cap,
			ms.mod_caplvl,
			ms.modi_cap,
			ms.modi_caplvl
		)
		IS DISTINCT FROM
		(
			$2,
			$3,
			$4,
			$5
		)
RETURNING -- inserted or modified
	0

-- :name mod_set_mod_priv_group
-- args: <pubkey> <group> <capabilities> <delpriv>
INSERT INTO
	ib.modsets AS ms
	(
		mod_pubkey,
		mod_group,

		mod_cap,
		mod_caplvl,
		modi_cap,
		modi_caplvl
	)
VALUES
	(
		$1,
		$2,

		$3,
		$4,
		$5,
		$6
	)
ON CONFLICT (mod_pubkey,mod_group)
WHERE
	b_id IS NULL AND b_p_id IS NULL AND
		mod_group IS NOT NULL
DO UPDATE
	SET
		mod_cap     = $3,
		mod_caplvl  = $4,
		modi_cap    = $5,
		modi_caplvl = $6
	WHERE
		(
			ms.mod_cap,
			ms.mod_caplvl,
			ms.modi_cap,
			ms.modi_caplvl
		)
		IS DISTINCT FROM
		(
			$3,
			$4,
			$5,
			$6
		)
RETURNING -- inserted or modified
	0


-- :name mod_unset_mod
-- args: <pubkey>
DELETE FROM
	ib.modsets
WHERE
	mod_pubkey = $1 AND
		mod_group IS NULL AND
		b_id IS NULL AND
		b_p_id IS NULL

-- :namet mod_fetch_and_clear_mod_msgs_common_a
WITH
	-- fetch messages
	zbp AS (
		SELECT
			b_id,
			b_p_id,
			b_t_id,
			date_sent,
			g_p_id
		FROM
			ib.bposts
-- :namet mod_fetch_and_clear_mod_msgs_common_b
		ORDER BY
			date_sent DESC,
			g_p_id DESC,
			b_id DESC
		LIMIT
			$2
	),
	-- delete from banlist
	zd AS (
		DELETE FROM
			ib.banlist bl
		USING
			zbp
		WHERE
			(bl.b_id,bl.b_p_id) = (zbp.b_id,zbp.b_p_id)
	)

SELECT
	zbp.date_sent,
	zbp.g_p_id,
	zbp.b_id,
	zbp.b_p_id,

	yb.b_name,
	y_gp.msgid,
	yp_bp.msgid,

	y_gp.title,
	y_gp.message,
	y_gp.extras -> 'text_attach',
	y_gp_f.fname
FROM
	zbp

-- board
JOIN
	ib.boards yb
ON
	zbp.b_id = yb.b_id

-- global post
JOIN
	ib.gposts y_gp
ON
	zbp.g_p_id = y_gp.g_p_id

-- files of global post
LEFT JOIN LATERAL
	(
		SELECT
			xf.fname
		FROM
			ib.files xf
		WHERE
			y_gp.g_p_id = xf.g_p_id
		ORDER BY
			xf.f_id -- important
	) AS y_gp_f
ON
	TRUE

-- parent board post
LEFT JOIN
	ib.bposts yp_bp
ON
	-- only activates if zbp was child post
	zbp.b_t_id <> zbp.b_p_id AND
		(zbp.b_id,zbp.b_t_id) = (yp_bp.b_id,yp_bp.b_p_id)

-- :name mod_fetch_and_clear_mod_msgs_start
-- args: <modid> <limit>
{{ .mod_fetch_and_clear_mod_msgs_common_a }}
		WHERE
			mod_id = $1
{{ .mod_fetch_and_clear_mod_msgs_common_b }}

-- :name mod_fetch_and_clear_mod_msgs_continue
-- args: <modid> <limit> <off_pdate> <off_gpid> <off_bid>
{{ .mod_fetch_and_clear_mod_msgs_common_a }}
		WHERE
			mod_id = $1 AND
				(date_sent,g_p_id,b_id) < ($3,$4,$5)
{{ .mod_fetch_and_clear_mod_msgs_common_b }}



-- :name mod_load_files
-- loads all files w/ gpids for checking
SELECT
	fname,
	thumb,
	fsize,
	array_agg(DISTINCT g_p_id)
FROM
	ib.files
WHERE
	(fname,thumb,fsize) > ($1,$2,$3)
GROUP BY
	fname,
	thumb,
	fsize
ORDER BY
	fname,
	thumb,
	fsize
LIMIT
	200



-- :name mod_check_article_for_push
-- extract stuff important to know on push
-- input: msgid
SELECT
	g_p_id,
	date_recv IS NOT NULL,
	has_ph IS TRUE,
	ph_ban IS TRUE,
	ph_banpriv
FROM
	ib.gposts
WHERE
	msgid = $1

-- :name mod_delete_ph_for_push
-- incase there's only ph data in there, delete returning all of it
-- also recheck values and don't delete if there's inconsistency
DELETE FROM
	ib.gposts
WHERE
(
	g_p_id,
	date_recv IS NOT NULL,
	has_ph IS TRUE,
	ph_ban IS TRUE,
	ph_banpriv
) = (
	$1,
	FALSE,
	TRUE,
	$2,
	$3
)
RETURNING
	-- all of ph data
	ph_ban,
	ph_banpriv

-- :name mod_add_ph_after_push
UPDATE
	ib.gposts
SET
	ph_ban = $2,
	ph_banpriv = $3
WHERE
	g_p_id = $1
