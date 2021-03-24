
CREATE FUNCTION ib.banlist_after_ins() RETURNS TRIGGER
AS $$
BEGIN

	RAISE WARNING 'banlist_after_ins msgid <%> b_name %', NEW.msgid, NEW.b_name;

	IF NEW.bt_b_id IS NULL THEN

		WITH
			changeban AS (
				SELECT
					NEW.bt_msgid AS bt_msgid,
					COUNT(exibl.ban_id) <> 0 AS ph_ban,
					MIN(exibl.dpriv)         AS ph_banpriv
				FROM
					ib.banlist exibl
				WHERE
					NEW.bt_msgid = exibl.bt_msgid AND
                        exibl.bt_b_id IS NULL
				GROUP BY
					NEW.bt_msgid
			)

		INSERT INTO
			ib.gposts
			(
				msgid,
				ph_ban,
				ph_banpriv
			)

		SELECT
			bt_msgid,
			ph_ban,
			ph_banpriv
		FROM
			changeban AS cb

		ON CONFLICT (msgid)
		DO UPDATE
		SET
			ph_ban     = EXCLUDED.ph_ban,
			ph_banpriv = EXCLUDED.ph_banpriv;

	ELSE

		WITH
			changeban AS (
				SELECT
					NEW.bt_msgid,
					COUNT(exibl.ban_id) <> 0 AS ph_ban,
					MIN(exibl.dpriv)         AS ph_banpriv
				FROM
					ib.banlist exibl
				WHERE
					NEW.bt_msgid = exibl.bt_msgid AND
                        exibl.bt_b_id = NEW.bt_b_id
				GROUP BY
					NEW.bt_msgid
			)

		INSERT INTO
			-- remark: trigger will ensure b_p_id
			ib.bposts
			(
				msgid,
				b_id,

				ph_ban,
				ph_banpriv
			)

		SELECT
			cb.bt_msgid,
			cb.bt_b_id,

			cb.ph_ban,
			cb.ph_banpriv
		FROM
			changeban AS cb

		ON CONFLICT (msgid,b_id)
		DO UPDATE
		SET
			ph_ban     = EXCLUDED.ph_ban,
			ph_banpriv = EXCLUDED.ph_banpriv;

	END IF;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

-- iirc there are no updates of banlist only inserts by one and deletes by many
CREATE TRIGGER banlist_after_ins
AFTER INSERT
ON ib.banlist
FOR EACH ROW
EXECUTE PROCEDURE ib.banlist_after_ins();



-- to be ran AFTER delet from banlist
CREATE FUNCTION ib.banlist_after_del() RETURNS TRIGGER
AS $$
BEGIN
	-- deleted rows may have redundant message IDs, and also groups;
	-- first we should recalculate delete power stuffs for msgids affected,
	-- then we should look into updated rows and
	-- if theres no has_ph on them and no date_recv, nuke.
	-- note that nuking happens inside trigger
	WITH
		changebans AS (
			SELECT
				delbl.msgid,
				delbl.b_name,
				COUNT (exibl.ban_id) <> 0 AS ph_ban,
				MIN   (exibl.dpriv)       AS ph_banpriv
			FROM
				(
					SELECT DISTINCT
						msgid,
						b_name
					FROM
						oldrows
				) AS delbl
			LEFT JOIN
				ib.banlist exibl
			ON
				delbl.msgid = exibl.msgid AND
					delbl.b_name = exibl.b_name
			GROUP BY
				delbl.msgid,
				delbl.b_name
		),
		updgposts AS (
			UPDATE
				ib.gposts ugp
			SET
				ph_ban     = cb.ph_ban,
				ph_banpriv = cb.ph_banpriv
			FROM
				changebans cb
			WHERE
				(cb.msgid = ugp.msgid) AND
					(cb.b_name IS NULL)
		)
	UPDATE
		ib.bposts ubp
	SET
		ph_ban     = cb.ph_ban,
		ph_banpriv = cb.ph_banpriv
	FROM
		changebans cb,
		ib.boards xb
	WHERE
		(cb.msgid = ubp.msgid) AND
			(cb.b_name = xb.b_name) AND
			(xb.b_id = ubp.b_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

-- iirc there are no updates of banlist only inserts by one and deletes by many
CREATE TRIGGER after_del
AFTER DELETE
ON ib.banlist
REFERENCING OLD TABLE AS oldrows
FOR EACH STATEMENT
EXECUTE PROCEDURE ib.banlist_after_del();
