
CREATE FUNCTION ib.modlist_gc() RETURNS TRIGGER
AS $$
DECLARE
	m_hasrefs BOOLEAN;
BEGIN

	SELECT
		COUNT(*) <> 0 AS hasrefs
	INTO
		m_hasrefs
	FROM
		ib.bposts
	WHERE
		mod_id = NEW.mod_id;

	IF (NOT m_hasrefs) THEN

		-- delet instead
		DELETE FROM
			ib.modlist mods
		WHERE
			mods.mod_id = NEW.mod_id;

		-- don't update
		RETURN NULL;

	END IF;

	RETURN NEW;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER gc
BEFORE UPDATE
ON ib.modlist
FOR EACH ROW
WHEN (NEW.automanage AND NOT OLD.automanage)
EXECUTE PROCEDURE ib.modlist_gc();



CREATE FUNCTION ib.modlist_changepriv() RETURNS TRIGGER
AS $$
BEGIN

	RAISE WARNING 'modlist_changepriv OP % mod_id %', TG_OP, NEW.mod_id;

	INSERT INTO
		ib.modlist_changes (
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
$$ LANGUAGE plpgsql;

-- if delete, then there are no posts to invoke by now
-- if insert, then there are no posts to invoke yet
CREATE TRIGGER changepriv
AFTER
	-- NOTE: inserts aren't logged, because no new messages would be awakened by them
	UPDATE OF
		mod_cap,
		mod_caplvl,
		mod_bcap,
		mod_bcaplvl,

		modi_cap,
		modi_caplvl,
		modi_bcap,
		modi_bcaplvl

ON ib.modlist
FOR EACH ROW
WHEN
	(
		-- could put it all in one IS DISTINCT FROM but that'd be too long
		(
			(OLD.mod_cap,OLD.mod_caplvl,OLD.mod_bcap,OLD.mod_bcaplvl)
							IS DISTINCT FROM
			(NEW.mod_cap,NEW.mod_caplvl,NEW.mod_bcap,NEW.mod_bcaplvl)
		)
		OR
		(
			(OLD.modi_cap,OLD.modi_caplvl,OLD.modi_bcap,OLD.modi_bcaplvl)
							IS DISTINCT FROM
			(NEW.modi_cap,NEW.modi_caplvl,NEW.modi_bcap,NEW.modi_bcaplvl)
		)
	)
EXECUTE PROCEDURE
	ib.modlist_changepriv();
