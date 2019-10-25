-- :name mod_joblist_modlist_changes_get
-- XXX we should somehow merge later ones into current one if we find them
SELECT
	j_id,

	mod_id,

	t_date_sent,
	t_g_p_id,
	t_b_id
FROM
	ib0.modlist_changes
ORDER BY
	j_id ASC
LIMIT
	1
FOR UPDATE

-- :name mod_joblist_modlist_changes_set
UPDATE
	ib0.modlist_changes
SET
	t_date_sent = $2,
	t_g_p_id    = $3,
	t_b_id      = $4
WHERE
	j_id = $1

-- :name mod_joblist_modlist_changes_del
DELETE FROM
	ib0.modlist_changes
WHERE
	j_id = $1



-- :name mod_joblist_refs_recalc_get
SELECT
	j_id,

	p_name,
	b_name,
	msgid,

	b_id,
	b_p_id
FROM
	ib0.refs_recalc
ORDER BY
	j_id ASC
LIMIT
	1
FOR UPDATE
-- :name mod_joblist_refs_recalc_set
UPDATE
	ib0.refs_recalc
SET
	b_id   = $2,
	b_p_id = $3
WHERE
	j_id = $1
-- :name mod_joblist_refs_recalc_del
DELETE FROM
	ib0.refs_recalc
WHERE
	j_id = $1
