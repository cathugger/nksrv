-- :name init_triggers_threads



-- :next
CREATE FUNCTION
	ib0.threads_fix_btid() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib0.boards
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
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER fix_btid
BEFORE INSERT
ON ib0.threads
FOR EACH ROW
WHEN (NEW.b_t_id IS NULL)
EXECUTE PROCEDURE ib0.threads_fix_btid()



-- :next
CREATE FUNCTION
	ib0.threads_ins_board_count() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib0.boards
	SET
		t_count = t_count + 1
	WHERE
		b_id = NEW.b_id;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER ins_board_count
AFTER INSERT
ON ib0.threads
FOR EACH ROW
EXECUTE PROCEDURE ib0.threads_ins_board_count()



-- :next
CREATE FUNCTION ib0.threads_del_board_count() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib0.boards
	SET
		t_count = t_count - 1
	WHERE
		b_id = OLD.b_id;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER del_board_count
AFTER DELETE
ON ib0.threads
FOR EACH ROW
EXECUTE PROCEDURE ib0.threads_del_board_count()
