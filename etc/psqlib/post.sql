

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
-- :namet post_template_newthread_ut
	ut AS (
		INSERT INTO
			ib0.threads (
				b_id,
				g_t_id,
				b_t_name,
				bump,
				skip_over
			)
		SELECT
			$12,        -- b_id
			ugp.g_p_id, -- g_t_id
			$13,        -- b_t_name
			$1,         -- date_sent
			$14         -- skip_over
		FROM
			ugp
		RETURNING
			b_id,
			b_t_id
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

-- :namet post_template_newpost_ubp_sb
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




-- :namet bepis
WITH
	
,
	ubp AS (
		INSERT INTO
			ib0.bposts (
				b_id,
				b_t_id,
				b_p_id,
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
		SELECT
			$12,           -- b_id
			ut.b_t_id,     -- b_t_id
			ut.b_t_id,     -- b_p_id
			$13,           -- p_name
			ugp.g_p_id,    -- g_p_id
			$3,            -- msgid
			ugp.date_sent, -- date_sent
			ugp.date_recv, -- date_recv
			FALSE,         -- sage
			ugp.f_count,   -- f_count
			$15,           -- mod_id
			$16            -- attrib
		FROM
			ut
		CROSS JOIN
			ugp
		RETURNING
			g_p_id,
			b_p_id
	)
