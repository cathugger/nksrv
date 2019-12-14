-- :name init_triggers_boards


-- :next
CREATE FUNCTION
	ib0.boards_prune_threads() RETURNS TRIGGER
AS $$
BEGIN

	-- delete excess ones
	DELETE FROM
		ib0.threads
	WHERE
		b_id = NEW.b_id AND t_order > NEW.cfg_t_thread_limit;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER prune_threads
AFTER UPDATE OF t_count
ON ib0.boards
FOR EACH ROW
WHEN (NEW.t_count > OLD.t_count AND NEW.t_count > NEW.cfg_t_thread_limit)
EXECUTE PROCEDURE ib0.boards_prune_threads()
