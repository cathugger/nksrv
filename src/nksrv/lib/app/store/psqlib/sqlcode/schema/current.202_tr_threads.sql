CREATE FUNCTION
	ib.threads_fix_btid() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib.boards
	SET
		last_id = last_id + 1
	WHERE
		b_id = NEW.b_id
	RETURNING
		last_id
	INTO STRICT
		NEW.b_t_id;

	RETURN NEW;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fix_btid
BEFORE INSERT
ON ib.threads
FOR EACH ROW
WHEN (NEW.b_t_id IS NULL)
EXECUTE PROCEDURE ib.threads_fix_btid();



CREATE FUNCTION
	ib.threads_ins_board_count() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib.boards
	SET
		t_count = t_count + 1
	WHERE
		b_id = NEW.b_id;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ins_board_count
AFTER INSERT
ON ib.threads
FOR EACH ROW
EXECUTE PROCEDURE ib.threads_ins_board_count();



CREATE FUNCTION ib.threads_del_board_count() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib.boards
	SET
		t_count = t_count - 1
	WHERE
		b_id = OLD.b_id;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER del_board_count
AFTER DELETE
ON ib.threads
FOR EACH ROW
EXECUTE PROCEDURE ib.threads_del_board_count();
