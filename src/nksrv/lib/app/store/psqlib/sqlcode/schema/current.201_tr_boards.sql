
CREATE FUNCTION
	ib.boards_prune_threads() RETURNS TRIGGER
AS $$
BEGIN

	-- delete excess ones
	DELETE FROM
		ib.threads
	WHERE
		b_id = NEW.b_id AND t_order > NEW.cfg_t_thread_limit;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prune_threads
AFTER UPDATE OF t_count
ON ib.boards
FOR EACH ROW
WHEN (NEW.t_count > OLD.t_count AND NEW.t_count > NEW.cfg_t_thread_limit)
EXECUTE PROCEDURE ib.boards_prune_threads();