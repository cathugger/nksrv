-- :name init_triggers_files



-- :next
CREATE FUNCTION ib0.files_after_insert() RETURNS TRIGGER
AS $$
BEGIN

	INSERT INTO
		ib0.files_uniq_fname AS uf (fname,cnt)
	SELECT
		fname,1
	FROM
		newrows
	ON CONFLICT (fname)
	DO UPDATE
	SET
		cnt = uf.cnt + 1;

	INSERT INTO
		ib0.files_uniq_thumb AS ut (fname,thumb,cnt)
	SELECT
		fname,thumb,1
	FROM
		newrows
	WHERE
		thumb <> ''
	ON CONFLICT (fname,thumb)
	DO UPDATE
	SET
		cnt = ut.cnt + 1;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER after_insert
AFTER INSERT
ON ib0.files
REFERENCING NEW TABLE AS newrows
FOR EACH STATEMENT
EXECUTE PROCEDURE ib0.files_after_insert()




-- :next
CREATE FUNCTION ib0.files_after_delete() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib0.files_uniq_fname uf
	SET
		cnt = uf.cnt - 1
	FROM
		oldrows
	WHERE
		uf.fname = oldrows.fname;

	UPDATE
		ib0.files_uniq_thumb ut
	SET
		cnt = ut.cnt - 1
	FROM
		oldrows
	WHERE
		oldrows.thumb <> '' AND
			(ut.fname,ut.thumb) = (oldrows.fname,oldrows.thumb);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER after_delete
AFTER DELETE
ON ib0.files
REFERENCING OLD TABLE AS oldrows
FOR EACH STATEMENT
EXECUTE PROCEDURE ib0.files_after_delete()


-- :next
CREATE FUNCTION ib0.files_uniq_fname_update_delete() RETURNS TRIGGER
AS $$
BEGIN

	-- "If you have no specific reason to make a trigger BEFORE or AFTER, the BEFORE case is more efficient, since the information about the operation doesn't have to be saved until end of statement."

	IF NEW.cnt < 0 THEN
		RAISE EXCEPTION 'counter went negative';
	END IF;

	-- mark to delet
	INSERT INTO
		ib0.files_deleted (fname)
	VALUES
		(NEW.fname);

	-- notify reaper
	NOTIFY ib0_files_deleted;

	-- proceed
	RETURN NEW;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER update_delete
BEFORE UPDATE
ON ib0.files_uniq_fname
FOR EACH ROW
WHEN (NEW.cnt <= 0)
EXECUTE PROCEDURE ib0.files_uniq_fname_update_delete()




-- :next
CREATE FUNCTION ib0.files_uniq_thumb_update_delete() RETURNS TRIGGER
AS $$
BEGIN

	-- "If you have no specific reason to make a trigger BEFORE or AFTER, the BEFORE case is more efficient, since the information about the operation doesn't have to be saved until end of statement."

	IF NEW.cnt < 0 THEN
		RAISE EXCEPTION 'counter went negative';
	END IF;

	-- mark to delet
	INSERT INTO
		ib0.fthumbs_deleted (fname,thumb)
	VALUES
		(NEW.fname,NEW.thumb);

	-- notify reaper
	NOTIFY ib0_fthumbs_deleted;

	-- proceed
	RETURN NEW;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER update_delete
BEFORE UPDATE
ON ib0.files_uniq_thumb
FOR EACH ROW
WHEN (NEW.cnt <= 0)
EXECUTE PROCEDURE ib0.files_uniq_thumb_update_delete()
