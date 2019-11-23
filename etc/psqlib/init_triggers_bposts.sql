-- :name init_triggers_bposts



-- :next
CREATE FUNCTION
	ib0.bposts_after_real_insert() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib0.boards
	SET
		p_count = p_count + 1
	WHERE
		b_id = NEW.b_id;

	UPDATE
		ib0.threads
	SET
		p_count = p_count + 1,
		f_count = f_count + NEW.f_count,
		fr_count = fr_count + ((NEW.f_count > 0 AND NEW.b_p_id != NEW.b_t_id) :: INTEGER)
	WHERE
		(b_id,b_t_id) = (NEW.b_id,NEW.b_t_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER x10_after_real_insert
AFTER INSERT
ON ib0.bposts
FOR EACH ROW
WHEN (NEW.date_recv IS NOT NULL)
EXECUTE PROCEDURE ib0.bposts_after_real_insert()



-- :next
CREATE FUNCTION
	ib0.bposts_after_real_delete() RETURNS TRIGGER
AS $$
BEGIN

	-- correct post count in board
	UPDATE
		ib0.boards
	SET
		p_count = p_count - 1
	WHERE
		b_id = OLD.b_id;

	-- correct post count in thread
	UPDATE
		ib0.threads
	SET
		p_count = p_count - 1,
		f_count = f_count - OLD.f_count,
		fr_count = fr_count - ((OLD.f_count > 0 AND OLD.b_p_id != OLD.b_t_id) :: INTEGER)
	WHERE
		(b_id,b_t_id) = (OLD.b_id,OLD.b_t_id);

	-- ref recalc will want this
	INSERT INTO
		t_del_bposts
		(
			b_id,
			msgid,
			p_name
		)
	VALUES
		(
			OLD.b_id,
			OLD.msgid,
			OLD.p_name
		);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER x10_after_real_delete
AFTER DELETE
ON ib0.bposts
FOR EACH ROW
WHEN (OLD.date_recv IS NOT NULL)
EXECUTE PROCEDURE ib0.bposts_after_real_delete()




-- :next
CREATE FUNCTION
	ib0.bposts_delete_modid() RETURNS TRIGGER
AS $$
BEGIN

	INSERT INTO t_del_modids (mod_id) VALUES (OLD.mod_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql


-- :next
CREATE TRIGGER x20_delete_modid
AFTER DELETE
ON ib0.bposts
FOR EACH ROW
WHEN (OLD.mod_id IS NOT NULL)
EXECUTE PROCEDURE ib0.bposts_delete_modid()




-- :next
CREATE FUNCTION
	ib0.bposts_thread_insert_bump() RETURNS TRIGGER
AS $$
BEGIN

	PERFORM ib0.bposts_thread_bump(NEW.b_id,NEW.b_t_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER x20_thread_insert_bump
AFTER INSERT
ON ib0.bposts
FOR EACH ROW
WHEN
	-- don't exec if it's OP - it should already have right value in that case
	(NEW.date_sent IS NOT NULL AND NEW.b_t_id != NEW.b_p_id)
EXECUTE PROCEDURE
	ib0.bposts_thread_insert_bump()


-- :next
CREATE FUNCTION
	ib0.bposts_thread_update_bump() RETURNS TRIGGER
AS $$
BEGIN

	PERFORM ib0.bposts_thread_bump(NEW.b_id,NEW.b_t_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
-- NOTE: not designed for post transfers between threads
CREATE TRIGGER x20_thread_update_bump
AFTER UPDATE OF
	date_sent
ON ib0.bposts
FOR EACH ROW
WHEN
	(OLD.date_sent IS DISTINCT FROM NEW.date_sent)
EXECUTE PROCEDURE
	ib0.bposts_thread_update_bump()


-- :next
CREATE FUNCTION
	ib0.bposts_thread_delete_bump() RETURNS TRIGGER
AS $$
BEGIN

	PERFORM ib0.bposts_thread_bump(OLD.b_id,OLD.b_t_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER x20_thread_bump
AFTER DELETE
ON ib0.bposts
FOR EACH ROW
WHEN
	-- don't exec if it's OP - it should already be nuked in that case
	(OLD.date_sent IS NOT NULL AND OLD.b_t_id != OLD.b_p_id)
EXECUTE PROCEDURE
	ib0.bposts_thread_delete_bump()




-- :next
CREATE FUNCTION ib0.bposts_fix_bpid() RETURNS TRIGGER
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
		NEW.b_p_id;

	RETURN NEW;

END;
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER fix_bpid
BEFORE INSERT
ON ib0.bposts
FOR EACH ROW
WHEN (NEW.b_p_id IS NULL)
EXECUTE PROCEDURE ib0.bposts_fix_bpid()





-- :next
CREATE FUNCTION ib0.bposts_gc_gposts() RETURNS TRIGGER
AS $$
BEGIN

	WITH
		delgp AS (
			DELETE FROM
				ib0.gposts gp
			USING
				(
					SELECT
						COUNT(*) <> 0 AS hasrefs
					FROM
						ib0.bposts
					WHERE
						g_p_id = OLD.g_p_id
				) AS x
			WHERE
				-- if it's this one and has actual content and has no more refs
				(gp.g_p_id = OLD.g_p_id) AND
					(gp.date_recv IS NOT NULL) AND
					(NOT x.hasrefs)
			RETURNING
				gp.has_ph,
				gp.ph_ban,
				gp.ph_banpriv
		)
	-- reinsert if it had ph data
	INSERT INTO
		ib0.gposts
		(
			has_ph,
			ph_ban,
			ph_banpriv
		)
	SELECT
		has_ph,
		ph_ban,
		ph_banpriv
	FROM
		delgp
	WHERE
		has_ph IS TRUE;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
-- TODO statement-wide
CREATE TRIGGER gc_gposts
AFTER DELETE
ON ib0.bposts
FOR EACH ROW
WHEN (OLD.g_p_id IS NOT NULL)
EXECUTE PROCEDURE ib0.bposts_gc_gposts()



-- :next
CREATE FUNCTION ib0.bposts_gc_modlist() RETURNS TRIGGER
AS $$
BEGIN

	DELETE FROM
		ib0.modlist mods
	USING
		(
			SELECT
				COUNT(*) <> 0 AS hasrefs
			FROM
				ib0.bposts
			WHERE
				mod_id = OLD.mod_id
		) AS x
	WHERE
		(mods.mod_id = OLD.mod_id) AND
			(mods.automanage) AND
			(NOT x.hasrefs);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql
-- :next
-- TODO per stmt
CREATE TRIGGER gc_modlist
AFTER DELETE
ON ib0.bposts
FOR EACH ROW
WHEN (OLD.mod_id IS NOT NULL)
EXECUTE PROCEDURE ib0.bposts_gc_modlist()





-- :next
CREATE FUNCTION ib0.bposts_ch_dpriv() RETURNS TRIGGER
AS $$
DECLARE
	m_g_p_id BIGINT;
BEGIN

	IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
		m_g_p_id := NEW.g_p_id;
	ELSIF TG_OP = 'DELETE' THEN
		m_g_p_id := OLD.g_p_id;
	END IF;

	UPDATE
		ib0.gposts gp
	SET
		gp.mod_dpriv = x.mod_dpriv
	FROM
		(
			SELECT
				MIN(mod_u_caplvl[1]) AS mod_dpriv
			FROM
				ib0.bposts
			WHERE
				g_p_id = m_g_p_id
		) AS x
	WHERE
		gp.g_p_id = m_g_p_id;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql

-- :next
CREATE TRIGGER ch_dpriv_ins
AFTER INSERT
ON ib0.bposts
FOR EACH ROW
WHEN (NEW.mod_u_caplvl[1] IS NOT NULL)
EXECUTE PROCEDURE ib0.bposts_ch_dpriv()

-- :next
CREATE TRIGGER ch_dpriv_upd
AFTER UPDATE OF mod_u_caplvl
ON ib0.bposts
FOR EACH ROW
WHEN (
	OLD.mod_u_caplvl[1] IS DISTINCT FROM NEW.mod_u_caplvl[1]
)
EXECUTE PROCEDURE ib0.bposts_ch_dpriv()

-- :next
CREATE TRIGGER ch_dpriv_del
AFTER DELETE
ON ib0.bposts
FOR EACH ROW
WHEN (OLD.mod_u_caplvl[1] IS NOT NULL)
EXECUTE PROCEDURE ib0.bposts_ch_dpriv()





-- :next
CREATE FUNCTION ib0.bposts_before_insert() RETURNS TRIGGER
AS $$
BEGIN

	-- calculate has_ph field
	-- I really want psql12's generated columns
	NEW.has_ph = ib0.calc_bpost_has_ph(NEW);

	RETURN NEW;

END;
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER before_insert
BEFORE INSERT
ON ib0.bposts
FOR EACH ROW
EXECUTE PROCEDURE ib0.bposts_before_insert()



-- :next
CREATE FUNCTION ib0.bposts_before_update() RETURNS TRIGGER
AS
$$
BEGIN

	-- calculate has_ph field
	-- I really want psql12's generated columns
	NEW.has_ph = ib0.calc_bpost_has_ph(NEW);

	-- magix: 0 - can ban >=0, 1 - can ban >=2, 2 - can ban >=2...
	IF (NEW.date_recv IS NOT NULL) AND
		(NEW.ph_ban IS TRUE) AND
		((NEW.mod_dpriv IS NULL) OR
			(NEW.mod_dpriv >= ((NEW.ph_banpriv + 1) & ~1)))
	THEN

		-- invoke delete to rid of shit what depends on this
		DELETE FROM
			ib0.bposts
		WHERE
			(b_id,b_p_id) = (NEW.b_id,NEW.b_p_id);

		-- now reinsert as we do have shit to store
		-- XXX we cound reuse same b_p_id but it sorta feels cleaner not to
		-- XXX that'll waste more b_p_ids tho
		INSERT INTO
			ib0.bposts
			(
				b_id,
				msgid,

				has_ph,
				ph_ban,
				ph_banpriv
			)
		VALUES
			(
				NEW.b_id,
				NEW.msgid,

				NEW.has_ph,
				NEW.ph_ban,
				NEW.ph_banpriv
			);

		-- early return to cancel UPDATE
		RETURN NULL;

	END IF;

	-- incase it's content-free now
	IF (NEW.date_recv IS NULL) AND (NEW.has_ph IS NOT TRUE) THEN

		DELETE FROM
			ib0.bposts
		WHERE
			(b_id,b_p_id) = (NEW.b_id,NEW.b_p_id);

		-- early return to cancel UPDATE
		RETURN NULL;

	END IF;

	RETURN NEW;

END;
$$ LANGUAGE plpgsql
-- :next
CREATE TRIGGER before_update
BEFORE UPDATE
ON ib0.bposts
FOR EACH ROW
EXECUTE PROCEDURE ib0.bposts_before_update()
