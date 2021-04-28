CREATE FUNCTION
	ib.bposts_after_real_insert() RETURNS TRIGGER
AS $$
BEGIN

	UPDATE
		ib.boards
	SET
		p_count = p_count + 1
	WHERE
		b_id = NEW.b_id;

	UPDATE
		ib.threads
	SET
		p_count = p_count + 1,
		f_count = f_count + NEW.f_count,
		fr_count = fr_count +
			((NEW.f_count > 0 AND NEW.b_p_id != NEW.b_t_id) :: INTEGER)
	WHERE
		(b_id,b_t_id) = (NEW.b_id,NEW.b_t_id);

	-- recalc refs for this bpost
	PERFORM ib.bposts_recalc_refs(NEW);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER x10_after_real_insert
AFTER INSERT
ON ib.bposts
FOR EACH ROW
WHEN (NEW.date_recv IS NOT NULL)
EXECUTE PROCEDURE ib.bposts_after_real_insert();



CREATE FUNCTION
	ib.bposts_after_real_delete() RETURNS TRIGGER
AS $$
BEGIN

	-- correct post count in board
	UPDATE
		ib.boards
	SET
		p_count = p_count - 1
	WHERE
		b_id = OLD.b_id;

	-- correct post count in thread
	UPDATE
		ib.threads
	SET
		p_count = p_count - 1,
		f_count = f_count - OLD.f_count,
		fr_count = fr_count -
			((OLD.f_count > 0 AND OLD.b_p_id != OLD.b_t_id) :: INTEGER)
	WHERE
		(b_id,b_t_id) = (OLD.b_id,OLD.b_t_id);

	-- recalc refs for this bpost
	PERFORM ib.bposts_recalc_refs(OLD);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER x10_after_real_delete
AFTER DELETE
ON ib.bposts
FOR EACH ROW
WHEN (OLD.date_recv IS NOT NULL)
EXECUTE PROCEDURE ib.bposts_after_real_delete();




CREATE FUNCTION
	ib.bposts_delete_modid() RETURNS TRIGGER
AS $$
BEGIN

	INSERT INTO t_del_modids (mod_id) VALUES (OLD.mod_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER x20_delete_modid
AFTER DELETE
ON ib.bposts
FOR EACH ROW
WHEN (OLD.mod_id IS NOT NULL)
EXECUTE PROCEDURE ib.bposts_delete_modid();




CREATE FUNCTION
	ib.bposts_thread_insert_bump() RETURNS TRIGGER
AS $$
BEGIN

	PERFORM ib.bposts_thread_bump(NEW.b_id,NEW.b_t_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER x20_thread_insert_bump
AFTER INSERT
ON ib.bposts
FOR EACH ROW
WHEN
	-- don't exec if it's OP - it should already have right value in that case
	(NEW.date_sent IS NOT NULL AND NEW.b_t_id != NEW.b_p_id)
EXECUTE PROCEDURE
	ib.bposts_thread_insert_bump();



CREATE FUNCTION
	ib.bposts_thread_update_bump() RETURNS TRIGGER
AS $$
BEGIN

	PERFORM ib.bposts_thread_bump(NEW.b_id,NEW.b_t_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

-- NOTE: not designed for post transfers between threads
CREATE TRIGGER x20_thread_update_bump
AFTER UPDATE OF
	date_sent
ON ib.bposts
FOR EACH ROW
WHEN
	(OLD.date_sent IS DISTINCT FROM NEW.date_sent)
EXECUTE PROCEDURE
	ib.bposts_thread_update_bump();



CREATE FUNCTION
	ib.bposts_thread_delete_bump() RETURNS TRIGGER
AS $$
BEGIN

	PERFORM ib.bposts_thread_bump(OLD.b_id,OLD.b_t_id);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER x20_thread_bump
AFTER DELETE
ON ib.bposts
FOR EACH ROW
WHEN
	-- don't exec if it's OP - it should already be nuked in that case
	(OLD.date_sent IS NOT NULL AND OLD.b_t_id != OLD.b_p_id)
EXECUTE PROCEDURE
	ib.bposts_thread_delete_bump();




CREATE FUNCTION ib.bposts_fix_bpid() RETURNS TRIGGER
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
		NEW.b_p_id;

	RETURN NEW;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fix_bpid
BEFORE INSERT
ON ib.bposts
FOR EACH ROW
WHEN (NEW.b_p_id IS NULL)
EXECUTE PROCEDURE ib.bposts_fix_bpid();



CREATE FUNCTION ib.bposts_gc_gposts() RETURNS TRIGGER
AS $$
BEGIN

	WITH
		delgp AS (
			DELETE FROM
				ib.gposts gp
			USING
				(
					SELECT
						COUNT(*) <> 0 AS hasrefs
					FROM
						ib.bposts
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
		ib.gposts
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
$$ LANGUAGE plpgsql;

-- TODO statement-wide
CREATE TRIGGER gc_gposts
AFTER DELETE
ON ib.bposts
FOR EACH ROW
WHEN (OLD.g_p_id IS NOT NULL)
EXECUTE PROCEDURE ib.bposts_gc_gposts();



CREATE FUNCTION ib.bposts_gc_modlist() RETURNS TRIGGER
AS $$
BEGIN

	DELETE FROM
		ib.modlist mods
	USING
		(
			SELECT
				COUNT(*) <> 0 AS hasrefs
			FROM
				ib.bposts
			WHERE
				mod_id = OLD.mod_id
		) AS x
	WHERE
		(mods.mod_id = OLD.mod_id) AND
			(mods.automanage) AND
			(NOT x.hasrefs);

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

-- TODO per stmt
CREATE TRIGGER gc_modlist
AFTER DELETE
ON ib.bposts
FOR EACH ROW
WHEN (OLD.mod_id IS NOT NULL)
EXECUTE PROCEDURE ib.bposts_gc_modlist();



CREATE FUNCTION ib.bposts_ch_dpriv() RETURNS TRIGGER
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
		ib.gposts gp
	SET
		gp.mod_dpriv = x.mod_dpriv
	FROM
		(
			SELECT
				MIN(mod_u_caplvl[1]) AS mod_dpriv
			FROM
				ib.bposts
			WHERE
				g_p_id = m_g_p_id
		) AS x
	WHERE
		gp.g_p_id = m_g_p_id;

	RETURN NULL;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ch_dpriv_ins
AFTER INSERT
ON ib.bposts
FOR EACH ROW
WHEN (NEW.mod_u_caplvl[1] IS NOT NULL)
EXECUTE PROCEDURE ib.bposts_ch_dpriv();

CREATE TRIGGER ch_dpriv_upd
AFTER UPDATE OF mod_u_caplvl
ON ib.bposts
FOR EACH ROW
WHEN (
	OLD.mod_u_caplvl[1] IS DISTINCT FROM NEW.mod_u_caplvl[1]
)
EXECUTE PROCEDURE ib.bposts_ch_dpriv();

CREATE TRIGGER ch_dpriv_del
AFTER DELETE
ON ib.bposts
FOR EACH ROW
WHEN (OLD.mod_u_caplvl[1] IS NOT NULL)
EXECUTE PROCEDURE ib.bposts_ch_dpriv();



CREATE FUNCTION ib.bposts_before_insert() RETURNS TRIGGER
AS $$
BEGIN

	-- calculate has_ph field
	-- I really want psql12's generated columns
	NEW.has_ph = ib.calc_bpost_has_ph(NEW);

	RETURN NEW;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER before_insert
BEFORE INSERT
ON ib.bposts
FOR EACH ROW
EXECUTE PROCEDURE ib.bposts_before_insert();



CREATE FUNCTION ib.bposts_before_update() RETURNS TRIGGER
AS
$$
BEGIN

	-- calculate has_ph field
	NEW.has_ph = ib.calc_bpost_has_ph(NEW);

	-- magix: 0 - can ban >=0, 1 - can ban >=2, 2 - can ban >=2...
	IF (NEW.date_recv IS NOT NULL) AND
		(NEW.ph_ban IS TRUE) AND
		((NEW.mod_dpriv IS NULL) OR
			(NEW.mod_dpriv >= ((NEW.ph_banpriv + 1) & ~1)))
	THEN

		-- invoke delete to rid of shit what depends on this
		DELETE FROM
			ib.bposts
		WHERE
			(b_id,b_p_id) = (NEW.b_id,NEW.b_p_id);

		-- now reinsert as we do have shit to store
		-- XXX we cound reuse same b_p_id but it sorta feels cleaner not to
		-- XXX that'll waste more b_p_ids tho
		INSERT INTO
			ib.bposts
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
			ib.bposts
		WHERE
			(b_id,b_p_id) = (NEW.b_id,NEW.b_p_id);

		-- early return to cancel UPDATE
		RETURN NULL;

	END IF;

	RETURN NEW;

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER before_update
BEFORE UPDATE
ON ib.bposts
FOR EACH ROW
EXECUTE PROCEDURE ib.bposts_before_update();
