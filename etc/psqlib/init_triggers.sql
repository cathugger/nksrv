-- :name init_triggers

-- :next
CREATE FUNCTION
	ib0.modlist_changepriv()
RETURNS
	TRIGGER
AS
$$
BEGIN

	INSERT INTO
		ib0.modlist_changes (
			mod_id,
			t_date_sent,
			t_g_p_id,
			t_b_id
		)
	VALUES
		(
			NEW.mod_id,
			NULL,
			NULL,
			NULL
		);

	-- poke process which can act upon it
	NOTIFY ib0_modlist_changes;

	RETURN NULL;

END;
$$
LANGUAGE
	plpgsql

-- :next
-- if delete, then there are no posts to invoke by now
-- if insert, then there are no posts to invoke yet
CREATE TRIGGER
	modlist_changepriv
AFTER
	-- NOTE: inserts aren't logged, because no new messages would be awakened by them
	UPDATE OF
		mod_cap,
		mod_bcap,
		mod_caplvl,
		mod_bcaplvl,
		modi_cap,
		modi_bcap,
		modi_caplvl,
		modi_bcaplvl
ON
	ib0.modlist
FOR EACH
	ROW
WHEN
	(
		(
			(OLD.mod_cap,OLD.mod_bcap,OLD.mod_caplvl,OLD.mod_bcaplvl)
			IS DISTINCT FROM
			(NEW.mod_cap,NEW.mod_bcap,NEW.mod_caplvl,NEW.mod_bcaplvl)
		)
		OR
		(
			(OLD.modi_cap,OLD.modi_bcap,OLD.modi_caplvl,OLD.modi_bcaplvl)
			IS DISTINCT FROM
			(NEW.modi_cap,NEW.modi_bcap,NEW.modi_caplvl,NEW.modi_bcaplvl)
		)
	)
EXECUTE PROCEDURE
	ib0.modlist_changepriv()



-- :next
-- to be ran AFTER delet from modsets
CREATE FUNCTION
	ib0.modsets_compute()
RETURNS
	TRIGGER
AS
$$
DECLARE
	pubkey   TEXT;
	r        RECORD;
	u_mod_id BIGINT;
BEGIN
	-- setup pubkey var
	IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
		pubkey := NEW.mod_pubkey;
	ELSIF TG_OP = 'DELETE' THEN
		pubkey := OLD.mod_pubkey;
	END IF;

	--RAISE WARNING 'OP % pubkey %', TG_OP, pubkey;

	-- recalc modlist val from modsets
	WITH
		comp_caps AS (
			SELECT
				mod_group,

				bit_or(mod_cap) AS calc_mod_cap,
				ARRAY[min(mod_caplvl[1])] AS calc_mod_caplvl,

				bit_or(modi_cap) AS calc_modi_cap,
				ARRAY[min(modi_caplvl[1])] AS calc_modi_caplvl

			FROM
				ib0.modsets
			WHERE
				mod_pubkey = pubkey
			GROUP BY
				mod_group
			ORDER BY
				mod_group
		)
	SELECT
		a.mod_cap,
		b.mod_bcap,
		c.mod_caplvl,
		d.mod_bcaplvl,

		ai.modi_cap,
		bi.modi_bcap,
		ci.modi_caplvl,
		di.modi_bcaplvl,

		z.automanage

	INTO STRICT
		r

	FROM
		(
			SELECT
				calc_mod_cap AS mod_cap
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS a
	FULL JOIN
		(
			SELECT
				jsonb_object(
					array_agg(mod_group),
					array_agg(calc_mod_cap::TEXT)) AS mod_bcap
			FROM
				comp_caps
			WHERE
				mod_group IS NOT NULL
		) AS b
	ON
		TRUE
	FULL JOIN
		(
			SELECT
				calc_mod_caplvl AS mod_caplvl
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS c
	ON
		TRUE
	FULL JOIN
		(
			SELECT
				jsonb_object(
					array_agg(mod_group),
					array_agg(calc_mod_caplvl::TEXT)) AS mod_bcaplvl
			FROM
				comp_caps
			WHERE
				mod_group IS NOT NULL AND
					calc_mod_caplvl IS NOT NULL
		) AS d
	ON
		TRUE

	FULL JOIN
		(
			SELECT
				calc_modi_cap AS modi_cap
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS ai
	ON
		TRUE
	FULL JOIN
		(
			SELECT
				jsonb_object(
					array_agg(mod_group),
					array_agg(calc_modi_cap::TEXT)) AS modi_bcap
			FROM
				comp_caps
			WHERE
				mod_group IS NOT NULL
		) AS bi
	ON
		TRUE
	FULL JOIN
		(
			SELECT
				calc_modi_caplvl AS modi_caplvl
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS ci
	ON
		TRUE
	FULL JOIN
		(
			SELECT
				jsonb_object(
					array_agg(mod_group),
					array_agg(calc_modi_caplvl::TEXT)) AS modi_bcaplvl
			FROM
				comp_caps
			WHERE
				mod_group IS NOT NULL AND
					calc_modi_caplvl IS NOT NULL
		) AS di
	ON
		TRUE

	FULL JOIN
		(
			SELECT
				COUNT(*) = 0 AS automanage
			FROM
				comp_caps
		) AS z
	ON
		TRUE;

	IF TG_OP = 'INSERT' THEN
		-- insert or update
		-- it may not exist yet
		-- or it may be automanaged
		INSERT INTO
			ib0.modlist (
				mod_pubkey,

				mod_cap,
				mod_bcap,
				mod_caplvl,
				mod_bcaplvl,

				modi_cap,
				modi_bcap,
				modi_caplvl,
				modi_bcaplvl,

				automanage
			)
		VALUES (
			pubkey,

			r.mod_cap,
			r.mod_bcap,
			r.mod_caplvl,
			r.mod_bcaplvl,

			r.modi_cap,
			r.modi_bcap,
			r.modi_caplvl,
			r.modi_bcaplvl,

			r.automanage
		)
		ON CONFLICT
			(mod_pubkey)
		DO UPDATE
			SET
				mod_cap     = EXCLUDED.mod_cap,
				mod_bcap    = EXCLUDED.mod_bcap,
				mod_caplvl  = EXCLUDED.mod_caplvl,
				mod_bcaplvl = EXCLUDED.mod_bcaplvl,

				modi_cap     = EXCLUDED.modi_cap,
				modi_bcap    = EXCLUDED.modi_bcap,
				modi_caplvl  = EXCLUDED.modi_caplvl,
				modi_bcaplvl = EXCLUDED.modi_bcaplvl,

				automanage = EXCLUDED.automanage;

	ELSIF TG_OP = 'UPDATE' THEN
		-- only update existing (because at this point it will exist)
		-- at this point it'll be automanaged too (because we're moding existing row)
		UPDATE
			ib0.modlist
		SET
			mod_cap     = r.mod_cap,
			mod_bcap    = r.mod_bcap,
			mod_caplvl  = r.mod_caplvl,
			mod_bcaplvl = r.mod_bcaplvl,

			modi_cap     = r.modi_cap,
			modi_bcap    = r.modi_bcap,
			modi_caplvl  = r.modi_caplvl,
			modi_bcaplvl = r.modi_bcaplvl

		WHERE
			mod_pubkey = pubkey;

	ELSIF TG_OP = 'DELETE' THEN
		-- update and possibly delete
		UPDATE
			ib0.modlist
		SET
			mod_cap     = r.mod_cap,
			mod_bcap    = r.mod_bcap,
			mod_caplvl  = r.mod_caplvl,
			mod_bcaplvl = r.mod_bcaplvl,

			modi_cap     = r.modi_cap,
			modi_bcap    = r.modi_bcap,
			modi_caplvl  = r.modi_caplvl,
			modi_bcaplvl = r.modi_bcaplvl,

			automanage  = r.automanage

		WHERE
			mod_pubkey = pubkey
		RETURNING
			mod_id
		INTO STRICT
			u_mod_id;

		IF r.automanage THEN
			-- if it's automanaged, do GC incase no post refers to it
			DELETE FROM
				ib0.modlist mods
			USING
				(
					SELECT
						mod_id,
						COUNT(*) <> 0 AS hasrefs
					FROM
						ib0.bposts
					WHERE
						mod_id = u_mod_id
					GROUP BY
						mod_id
				) AS rcnts
			WHERE
				mods.mod_id = rcnts.mod_id AND
					rcnts.hasrefs = FALSE;

		END IF;

	END IF;

	RETURN NULL;

END;
$$
LANGUAGE
	plpgsql

-- :next
CREATE TRIGGER
	modsets_compute
AFTER
	INSERT OR UPDATE OR DELETE
ON
	ib0.modsets
FOR EACH
	ROW
EXECUTE PROCEDURE
	ib0.modsets_compute()



-- :next
-- to be ran AFTER delet from banlist
CREATE FUNCTION
	ib0.banlist_after_del()
RETURNS
	TRIGGER
AS
$$
BEGIN
	-- garbage collect void placeholder posts when all bans for them are lifted
	DELETE FROM
		ib0.gposts xp
	USING
		(
			SELECT
				delbl.msgid,COUNT(exibl.msgid) > 0 AS hasrefs
			FROM
				oldrows AS delbl
			LEFT JOIN
				ib0.banlist exibl
			ON
				delbl.msgid = exibl.msgid
			WHERE
				delbl.msgid IS NOT NULL
			GROUP BY
				delbl.msgid
		) AS delp
	WHERE
		delp.hasrefs = FALSE AND
			delp.msgid = xp.msgid AND
			xp.date_recv IS NULL;

	RETURN NULL;
END;
$$
LANGUAGE
	plpgsql

-- :next
CREATE TRIGGER
	banlist_after_del
AFTER
	DELETE
ON
	ib0.banlist
REFERENCING
	OLD TABLE AS oldrows
FOR EACH
	STATEMENT
EXECUTE PROCEDURE
	ib0.banlist_after_del()
