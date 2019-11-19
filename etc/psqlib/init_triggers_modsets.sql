-- :name init_triggers_modsets



-- :next
-- to be ran AFTER delet from modsets
CREATE FUNCTION ib0.modsets_recompute() RETURNS TRIGGER
AS $$
DECLARE
	pubkey   TEXT;
	r        RECORD;
BEGIN

	-- setup pubkey var
	IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
		pubkey := NEW.mod_pubkey;
	ELSIF TG_OP = 'DELETE' THEN
		pubkey := OLD.mod_pubkey;
	END IF;


	RAISE WARNING 'modsets_compute OP % pubkey %', TG_OP, pubkey;



	-- recalc modlist val from modsets
	WITH

		comp_caps AS
		(
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
		b.mod_caplvl,
		c.mod_bcap,
		d.mod_bcaplvl,

		ai.modi_cap,
		bi.modi_caplvl,
		ci.modi_bcap,
		di.modi_bcaplvl,

		z.automanage


	INTO STRICT r

	FROM

		-- mod_cap
		(
			SELECT
				calc_mod_cap AS mod_cap
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS a

									FULL JOIN

		-- mod_caplvl
		(
			SELECT
				calc_mod_caplvl AS mod_caplvl
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS b

									ON TRUE

									FULL JOIN

		-- mod_bcap
		(
			SELECT
				jsonb_object(
					array_agg(mod_group),
					array_agg(calc_mod_cap::TEXT)) AS mod_bcap
			FROM
				comp_caps
			WHERE
				mod_group IS NOT NULL
		) AS c

									ON TRUE

									FULL JOIN

		-- mod_bcaplvl
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

									ON TRUE




									FULL JOIN

		-- modi_cap
		(
			SELECT
				calc_modi_cap AS modi_cap
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS ai

									ON TRUE

									FULL JOIN

		-- modi_caplvl
		(
			SELECT
				calc_modi_caplvl AS modi_caplvl
			FROM
				comp_caps
			WHERE
				mod_group IS NULL
		) AS bi

									ON TRUE

									FULL JOIN

		-- modi_bcap
		(
			SELECT
				jsonb_object(
					array_agg(mod_group),
					array_agg(calc_modi_cap::TEXT)) AS modi_bcap
			FROM
				comp_caps
			WHERE
				mod_group IS NOT NULL
		) AS ci

									ON TRUE

									FULL JOIN

		-- modi_bcaplvl
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

									ON TRUE




									FULL JOIN

		-- automanage
		(
			SELECT
				COUNT(*) = 0 AS automanage
			FROM
				comp_caps
		) AS z

									ON TRUE;






	IF TG_OP = 'INSERT' THEN

		-- insert or update
		-- it may not exist yet
		-- or it may be automanaged
		INSERT INTO
			ib0.modlist
			(
				mod_pubkey,

				mod_cap,
				mod_caplvl,
				mod_bcap,
				mod_bcaplvl,

				modi_cap,
				modi_caplvl,
				modi_bcap,
				modi_bcaplvl,

				automanage
			)

		VALUES
			(
				pubkey,

				r.mod_cap,
				r.mod_caplvl,
				r.mod_bcap,
				r.mod_bcaplvl,

				r.modi_cap,
				r.modi_caplvl,
				r.modi_bcap,
				r.modi_bcaplvl,

				r.automanage
			)

		ON CONFLICT (mod_pubkey)
		DO UPDATE
		SET
			mod_cap     = EXCLUDED.mod_cap,
			mod_caplvl  = EXCLUDED.mod_caplvl,
			mod_bcap    = EXCLUDED.mod_bcap,
			mod_bcaplvl = EXCLUDED.mod_bcaplvl,

			modi_cap     = EXCLUDED.modi_cap,
			modi_caplvl  = EXCLUDED.modi_caplvl,
			modi_bcap    = EXCLUDED.modi_bcap,
			modi_bcaplvl = EXCLUDED.modi_bcaplvl,

			automanage = EXCLUDED.automanage;


	ELSIF TG_OP = 'UPDATE' OR TG_OP = 'DELETE' THEN

		-- only update existing (because at this point it will exist)
		-- at this point it'll be automanaged too (because we're moding existing row)
		UPDATE
			ib0.modlist

		SET
			mod_cap     = r.mod_cap,
			mod_caplvl  = r.mod_caplvl,
			mod_bcap    = r.mod_bcap,
			mod_bcaplvl = r.mod_bcaplvl,

			modi_cap     = r.modi_cap,
			modi_caplvl  = r.modi_caplvl,
			modi_bcap    = r.modi_bcap,
			modi_bcaplvl = r.modi_bcaplvl,

			automanage  = r.automanage

		WHERE
			mod_pubkey = pubkey;

	END IF;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER recompute
AFTER INSERT OR UPDATE OR DELETE
ON ib0.modsets
FOR EACH ROW
EXECUTE PROCEDURE ib0.modsets_recompute()
