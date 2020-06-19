-- SQL statements used for posting
-- single post can be for new thread or new reply
-- single post can add to single or multiple boards
-- single post can have zero, one or many attachments
-- these categories are optimized looking at actual usage
-- it will result in 2×2×3 = 12 statements
--
-- this stuff uses templates a lot
--
-- * post_template_common_ugp
-- - for global post insertion into gposts table.
--   same fragment is used for all statements because structurically it's pretty much the same for all non-placeholder posts.
--
-- * post_template_newthread_ut_common_insert, post_template_newthread_ut_common_return
--   post_template_newthread_ut_sb, post_template_newthread_ut_mb
-- - insertion into threads table (single and multi board cases)
--
-- * post_template_common_ubp_insert, post_template_common_ubp_return
--   post_template_newthread_ubp, post_template_newreply_ubp_sb, post_template_newreply_ubp_mb
-- - insertion into bposts table
--
-- * post_template_common_uf_one, post_template_common_uf_many
-- - fragments for one and many files




-- :namet post_template_common_ugp
	ugp AS (
		INSERT INTO
			ib0.gposts (
				date_sent,     -- 1
				date_recv,     -- NOW()
				sage,          -- 2
				f_count,       -- 3
				msgid,         -- 4
				title,         -- 5
				author,        -- 6
				trip,          -- 7
				message,       -- 8
				headers,       -- 9
				attrib,        -- 10
				layout,        -- 11
				extras         -- 12
			)
		VALUES
			(
				$1,        -- date_sent
				NOW(),     -- date_recv
				$2,        -- sage
				$3,        -- f_count
				$4,        -- msgid
				$5,        -- title
				$6,        -- author
				$7,        -- trip
				$8,        -- message
				$9,        -- headers
				$10,       -- attrib
				$11,       -- layout
				$12        -- extras
			)
		RETURNING
			g_p_id,
			date_sent,
			date_recv,
			sage,
			f_count
	)
-- :namet post_template_newthread_ut_common_insert
		INSERT INTO
			ib0.threads (
				b_id,
				g_t_id,
				b_t_name,
				bump,
				skip_over
			)
-- :namet post_template_newthread_ut_common_return
		RETURNING
			b_id,
			b_t_id
-- :namet post_template_newthread_ut_sb
	ut AS (
{{ .post_template_newthread_ut_common_insert }}
		SELECT
			$12,        -- b_id
			ugp.g_p_id, -- g_t_id
			$13,        -- b_t_name
			$1,         -- date_sent
			$14         -- skip_over
		FROM
			ugp
{{ .post_template_newthread_ut_common_return }}
	)
-- :namet post_template_newthread_ut_mb
	ut AS (
{{ .post_template_newthread_ut_common_insert }}
		SELECT
			x.b_id,     -- b_id
			ugp.g_p_id, -- g_t_id
			x.b_t_name  -- b_t_name
			$1,         -- date_sent
			$14         -- skip_over
		FROM
			ugp
		CROSS JOIN
			UNNEST(
				$12,
				$13
			) AS x (
				b_id,
				b_t_name
			)
{{ .post_template_newthread_ut_common_return }}
	)

-- :namet post_template_common_ubp_insert
		INSERT INTO
			ib0.bposts (
				b_id,
				b_t_id,
				p_name,
				g_p_id,
				msgid,
				date_sent,
				date_recv,
				sage,
				f_count,
				mod_id,
				attrib
			)
-- :namet post_template_common_ubp_return
		RETURNING
			g_p_id,
			b_p_id

-- :namet post_template_newthread_ubp
	ubp AS (
{{ .post_template_common_ubp_insert }}
		SELECT
			ut.b_id,       -- b_id
			ut.b_t_id,     -- b_t_id
			ut.b_t_id,     -- b_p_id
			$14,           -- p_name
			ugp.g_p_id,    -- g_p_id
			$4,            -- msgid
			ugp.date_sent, -- date_sent
			ugp.date_recv, -- date_recv
			ugp.sage,      -- sage
			ugp.f_count,   -- f_count
			$16,           -- mod_id
			$17            -- attrib
		FROM
			ut
		CROSS JOIN
			ugp
{{ .post_template_common_ubp_return }}
	)
-- :namet post_template_newreply_ubp_sb
	ubp AS (
{{ .post_template_common_ubp_insert }}
		SELECT
			$13,           -- b_id
			$14,           -- b_t_id
			$15,           -- p_name
			ugp.g_p_id,    -- g_p_id
			$4,            -- msgid
			ugp.date_sent, -- date_sent
			ugp.date_recv, -- date_recv
			ugp.sage,      -- sage
			ugp.f_count,   -- f_count
			$16,           -- mod_id
			$17            -- attrib
		FROM
			ugp
{{ .post_template_common_ubp_return }}
	)
-- :namet post_template_newreply_ubp_mb
	ubp AS (
{{ .post_template_common_ubp_insert }}
		SELECT
			x.*,           -- b_id,b_t_id,p_name
			ugp.g_p_id,    -- g_p_id
			$4,            -- msgid
			ugp.date_sent, -- date_sent
			ugp.date_recv, -- date_recv
			ugp.sage,      -- sage
			ugp.f_count,   -- f_count
			$16,           -- mod_id
			$17            -- attrib
		FROM
			ugp
		CROSS JOIN
			UNNEST(
				$13,
				$14,
				$15
			) AS x
{{ .post_template_common_ubp_return }}
	)




-- :namet post_template_common_uf_one
	uf AS (
		INSERT INTO
			ib0.files (
				g_p_id,
				ftype,
				fsize,
				fname,
				thumb,
				oname,
				filecfg,
				thumbcfg,
				extras
			)
		SELECT
			g_p_id, -- g_p_id
			$18,    -- ftype
			$19,    -- fsize
			$20,    -- fname
			$21,    -- thumb
			$22,    -- oname
			$23,    -- filecfg
			$24,    -- thumbcfg
			$25     -- extras
		FROM
			ugp
	)
-- :namet post_template_common_uf_many
	uf AS (
		INSERT INTO
			ib0.files (
				g_p_id,
				ftype,
				fsize,
				fname,
				thumb,
				oname,
				filecfg,
				thumbcfg,
				extras
			)
		SELECT
			ugp.g_p_id, -- g_p_id
			x.*
		FROM
			ugp
		CROSS JOIN
			UNNEST(
				$18, -- ftype
				$19, -- fsize
				$20, -- fname
				$21, -- thumb
				$22, -- oname
				$23, -- filecfg
				$24, -- thumbcfg
				$25  -- extras
			) AS x
	)




-- :namet post_template_common_result
SELECT
	g_p_id,
	b_p_id
FROM
	ubp





-- :name post_newthread_sb_nf
-- single board, no files
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newthread_ut_sb }},
{{ .post_template_newthread_ubp }}
{{ .post_template_common_result }}
-- :name post_newthread_mb_nf
-- multi board, no files
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newthread_ut_mb }},
{{ .post_template_newthread_ubp }}
{{ .post_template_common_result }}

-- :name post_newthread_sb_sf
-- single board, single file
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newthread_ut_sb }},
{{ .post_template_newthread_ubp }},
{{ .post_template_common_uf_one }}
{{ .post_template_common_result }}
-- :name post_newthread_mb_sf
-- multi board, single file
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newthread_ut_mb }},
{{ .post_template_newthread_ubp }},
{{ .post_template_common_uf_one }}
{{ .post_template_common_result }}

-- :name post_newthread_sb_mf
-- single board, many files
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newthread_ut_sb }},
{{ .post_template_newthread_ubp }},
{{ .post_template_common_uf_many }}
{{ .post_template_common_result }}
-- :name post_newthread_mb_mf
-- multi board, many files
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newthread_ut_mb }},
{{ .post_template_newthread_ubp }},
{{ .post_template_common_uf_many }}
{{ .post_template_common_result }}



-- :name post_newreply_sb_nf
-- single board, no files
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newreply_ubp_sb }}
{{ .post_template_common_result }}
-- :name post_newreply_mb_nf
-- multi board, no files
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newreply_ubp_mb }}
{{ .post_template_common_result }}

-- :name post_newreply_sb_sf
-- single board, single file
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newreply_ubp_sb }},
{{ .post_template_common_uf_one }}
{{ .post_template_common_result }}
-- :name post_newreply_mb_sf
-- multi board, single file
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newreply_ubp_mb }},
{{ .post_template_common_uf_one }}
{{ .post_template_common_result }}

-- :name post_newreply_sb_mf
-- single board, many files
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newreply_ubp_sb }},
{{ .post_template_common_uf_many }}
{{ .post_template_common_result }}
-- :name post_newreply_mb_mf
-- multi board, many files
WITH
{{ .post_template_common_ugp }},
{{ .post_template_newreply_ubp_mb }},
{{ .post_template_common_uf_many }}
{{ .post_template_common_result }}
