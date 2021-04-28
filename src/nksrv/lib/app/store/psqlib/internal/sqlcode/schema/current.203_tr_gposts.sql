
CREATE FUNCTION ib.gposts_after_real_delete() RETURNS TRIGGER
AS $$
BEGIN

	INSERT INTO t_del_gposts (msgid) VALUES (OLD.msgid);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

-- TODO for each statement
CREATE TRIGGER after_real_delete
AFTER DELETE
ON ib.gposts
FOR EACH ROW
WHEN (OLD.date_recv IS NOT NULL)
EXECUTE PROCEDURE ib.gposts_after_real_delete();



CREATE FUNCTION ib.gposts_before_insert() RETURNS TRIGGER
AS $$
BEGIN

	RAISE WARNING 'gposts_before_insert <%> (date_recv: %)', NEW.msgid, NEW.date_recv;

	-- calculate has_ph field
	NEW.has_ph = ib.calc_gpost_has_ph(NEW);

	RETURN NEW;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER before_insert
BEFORE INSERT
ON ib.gposts
FOR EACH ROW
EXECUTE PROCEDURE ib.gposts_before_insert();



CREATE FUNCTION ib.gposts_before_update() RETURNS TRIGGER
AS
$$
BEGIN

	-- calculate has_ph field
	NEW.has_ph = ib.calc_gpost_has_ph(NEW);

	RAISE WARNING 'gposts_before_update <%>', NEW.msgid;

	-- magix: 0 - can ban >=0, 1 - can ban >=2, 2 - can ban >=2...
	IF (NEW.date_recv IS NOT NULL) AND
		(NEW.ph_ban IS TRUE) AND
		((NEW.mod_dpriv IS NULL) OR
			(NEW.mod_dpriv >= ((NEW.ph_banpriv + 1) & ~1)))
	THEN

		RAISE WARNING 'gposts_before_update <%> nuke existing', NEW.msgid;

		-- invoke delete to rid of shit what depends on this
		DELETE FROM
			ib.gposts
		WHERE
			g_p_id = NEW.g_p_id;

		-- now reinsert as we do have shit to store
		INSERT INTO
			ib.gposts
			(
				msgid,
				has_ph,
				ph_ban,
				ph_banpriv
			)
		VALUES
			(
				NEW.msgid,
				NEW.has_ph,
				NEW.ph_ban,
				NEW.ph_banpriv
			);

		-- early return to cancel UPDATE
		RETURN NULL;

	END IF;

	-- incase insert would be empty, nuke it
	IF (NEW.date_recv IS NULL) AND (NEW.has_ph IS NOT TRUE) THEN

		RAISE WARNING 'gposts_before_update <%> nuke empty', NEW.msgid;

		DELETE FROM
			ib.gposts
		WHERE
			g_p_id = NEW.g_p_id;

		-- early return to cancel UPDATE
		RETURN NULL;

	END IF;

	RETURN NEW;

END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER before_update
BEFORE UPDATE
ON ib.gposts
FOR EACH ROW
EXECUTE PROCEDURE ib.gposts_before_update();
