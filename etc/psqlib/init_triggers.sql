-- :name init_triggers



-- :next
CREATE FUNCTION
	ib0.calc_gpost_has_ph(r RECORD) RETURNS BOOLEAN
AS $$
BEGIN
	-- will change once more stuff is added
	RETURN r.ph_ban;
END;
$$ LANGUAGE plpgsql



-- :next
CREATE FUNCTION
	ib0.calc_bpost_has_ph(r RECORD) RETURNS BOOLEAN
AS $$
BEGIN
	-- will change once more stuff is added
	RETURN r.ph_ban;
END;
$$ LANGUAGE plpgsql



-- :next
CREATE FUNCTION
	ib0.bposts_thread_bump(x_b_id INTEGER, x_b_t_id BIGINT) RETURNS VOID
AS $$
BEGIN

	-- XXX could use OVER query maybe?
	UPDATE
		ib0.threads
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
						ib0.bposts
					WHERE
						-- count sages against bump limit.
						-- because others do it like that :<
						(b_id,b_t_id) = (x_b_id,x_b_t_id)
					ORDER BY
						date_sent ASC,
						b_p_id ASC
					LIMIT
						-- XXX can't do this in any prettier way?
						(SELECT cfg_t_bump_limit FROM ib0.boards WHERE b_id = x_b_id)
					-- take bump posts, sorted by original date,
					-- only upto bump limit
				) AS tt
			-- and pick last non-sage one
			WHERE
				sage IS NOT TRUE
			ORDER BY
				date_sent DESC,
				b_p_id DESC
			LIMIT
				1
		) as xbump
	WHERE
		(b_id,b_t_id) = (x_b_id,x_b_t_id);

END;
$$ LANGUAGE plpgsql
