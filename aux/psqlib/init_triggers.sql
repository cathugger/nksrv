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
		mod_dpriv,
		mod_bdpriv
ON
	ib0.modlist
FOR EACH
	ROW
WHEN
	((OLD.mod_cap,OLD.mod_bcap,OLD.mod_dpriv,OLD.mod_bdpriv)
		IS DISTINCT FROM
		(NEW.mod_cap,NEW.mod_bcap,NEW.mod_dpriv,NEW.mod_bdpriv))
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
				bit_or(mod_cap) AS mod_calccap,
				min(mod_dpriv) AS mod_calcdpriv
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
		c.mod_dpriv,
		d.mod_bdpriv,
		z.automanage
	INTO STRICT
		r
	FROM
		(
			SELECT
				mod_calccap AS mod_cap
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
					array_agg(mod_calccap::TEXT)) AS mod_bcap
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
				mod_calcdpriv AS mod_dpriv
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
					array_agg(mod_calcdpriv::TEXT)) AS mod_bdpriv
			FROM
				comp_caps
			WHERE
				mod_group IS NOT NULL AND
					mod_calcdpriv IS NOT NULL
		) AS d
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
				mod_dpriv,
				mod_bdpriv,
				automanage
			)
		VALUES (
			pubkey,
			r.mod_cap,
			r.mod_bcap,
			r.mod_dpriv,
			r.mod_bdpriv,
			r.automanage
		)
		ON CONFLICT
			(mod_pubkey)
		DO UPDATE
			SET
				mod_cap    = EXCLUDED.mod_cap,
				mod_bcap   = EXCLUDED.mod_bcap,
				mod_dpriv  = EXCLUDED.mod_dpriv,
				mod_bdpriv = EXCLUDED.mod_bdpriv,
				automanage = EXCLUDED.automanage;

	ELSIF TG_OP = 'UPDATE' THEN
		-- only update existing (because at this point it will exist)
		-- at this point it'll be automanaged too (because we're moding existing row)
		UPDATE
			ib0.modlist
		SET
			mod_cap    = r.mod_cap,
			mod_bcap   = r.mod_bcap,
			mod_dpriv  = r.mod_dpriv,
			mod_bdpriv = r.mod_bdpriv
		WHERE
			mod_pubkey = pubkey;

	ELSIF TG_OP = 'DELETE' THEN
		-- update and possibly delete
		UPDATE
			ib0.modlist
		SET
			mod_cap    = r.mod_cap,
			mod_bcap   = r.mod_bcap,
			mod_dpriv  = r.mod_dpriv,
			mod_bdpriv = r.mod_bdpriv,
			automanage = r.automanage
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
