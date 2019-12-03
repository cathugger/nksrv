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

	-- delete instead of updating
	DELETE FROM
		ib0.files_uniq_fname uf
	WHERE
		uf.fname = NEW.fname;

	-- mark that we deleted
	INSERT INTO
		t_del_files (fname)
	VALUES
		(NEW.fname);

	-- do not update
	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER update_delete
BEFORE UPDATE
ON ib0.files_uniq_fname
FOR EACH ROW
WHEN (NEW.cnt = 0)
EXECUTE PROCEDURE ib0.files_uniq_fname_update_delete()




-- :next
CREATE FUNCTION ib0.files_uniq_thumb_update_delete() RETURNS TRIGGER
AS $$
BEGIN

	-- delete instead of updating
	DELETE FROM
		ib0.files_uniq_thumb ut
	WHERE
		(ut.fname,ut.thumb) = (NEW.fname,NEW.thumb);

	-- mark that we deleted
	INSERT INTO
		t_del_fthumbs (fname,thumb)
	VALUES
		(NEW.fname,NEW.thumb);

	-- do not update
	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER update_delete
BEFORE UPDATE
ON ib0.files_uniq_thumb
FOR EACH ROW
WHEN (NEW.cnt = 0)
EXECUTE PROCEDURE ib0.files_uniq_thumb_update_delete()
