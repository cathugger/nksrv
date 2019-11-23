-- :name init_triggers_banlist



-- :next
CREATE FUNCTION ib0.banlist_after_ins() RETURNS TRIGGER
AS $$
BEGIN

	RAISE WARNING 'banlist_after_ins msgid <%> b_name %', NEW.msgid, NEW.b_name;

	IF NEW.b_name IS NULL THEN

		WITH
			changeban AS (
				SELECT
					NEW.msgid AS msgid,
					COUNT(exibl.ban_id) <> 0 AS ph_ban,
					MIN(exibl.dpriv)         AS ph_banpriv
				FROM
					ib0.banlist exibl
				WHERE
					NEW.msgid = exibl.msgid AND exibl.b_name IS NULL
				GROUP BY
					NEW.msgid
			)

		INSERT INTO
			ib0.gposts
			(
				msgid,
				ph_ban,
				ph_banpriv
			)

		SELECT
			msgid,
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
					NEW.msgid,
					COUNT(exibl.ban_id) <> 0 AS ph_ban,
					MIN(exibl.dpriv)         AS ph_banpriv
				FROM
					ib0.banlist exibl
				WHERE
					NEW.msgid = exibl.msgid AND exibl.b_name = NEW.b_name
				GROUP BY
					NEW.msgid
			),
			bb AS (
				SELECT
					b_id
				FROM
					ib0.boards
				WHERE
					b_name = NEW.b_name
			)

		INSERT INTO
			-- remark: trigger will ensure b_p_id
			ib0.bposts
			(
				msgid,
				b_id,

				ph_ban,
				ph_banpriv
			)

		SELECT
			cb.msgid,
			bb.b_id,

			cb.ph_ban,
			cb.ph_banpriv
		FROM
			changeban AS cb
		JOIN
			bb
		ON
			TRUE

		ON CONFLICT (msgid,b_id)
		DO UPDATE
		SET
			ph_ban     = EXCLUDED.ph_ban,
			ph_banpriv = EXCLUDED.ph_banpriv;

	END IF;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
-- iirc there are no updates of banlist only inserts by one and deletes by many
CREATE TRIGGER banlist_after_ins
AFTER INSERT
ON ib0.banlist
FOR EACH ROW
EXECUTE PROCEDURE ib0.banlist_after_ins()



-- :next
-- to be ran AFTER delet from banlist
CREATE FUNCTION ib0.banlist_after_del() RETURNS TRIGGER
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
				ib0.banlist exibl
			ON
				delbl.msgid = exibl.msgid AND
					delbl.b_name = exibl.b_name
			GROUP BY
				delbl.msgid,
				delbl.b_name
		),
		updgposts AS (
			UPDATE
				ib0.gposts ugp
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
		ib0.bposts ubp
	SET
		ph_ban     = cb.ph_ban,
		ph_banpriv = cb.ph_banpriv
	FROM
		changebans cb,
		ib0.boards xb
	WHERE
		(cb.msgid = ubp.msgid) AND
			(cb.b_name = xb.b_name) AND
			(xb.b_id = ubp.b_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
-- iirc there are no updates of banlist only inserts by one and deletes by many
CREATE TRIGGER after_del
AFTER DELETE
ON ib0.banlist
REFERENCING OLD TABLE AS oldrows
FOR EACH STATEMENT
EXECUTE PROCEDURE ib0.banlist_after_del()
