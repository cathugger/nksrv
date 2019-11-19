-- :name init_triggers_files



-- :next
CREATE FUNCTION ib0.files_after_delete() RETURNS TRIGGER
AS $$
BEGIN

	INSERT INTO
		t_del_files (fname)
	SELECT
		fname
	FROM
		(
			SELECT
				delfnames.fname,
				allfnames.cangc
			FROM
				(
					SELECT DISTINCT
						fname
					FROM
						oldrows
				) AS delfnames
			LEFT JOIN
				LATERAL (
					SELECT
						(COUNT(*) == 0) AS cangc
					FROM
						ib0.files xf
					WHERE
						delfnames.fname = xf.fname
				) AS allfnames
			ON
				TRUE
		) AS gcf
	WHERE
		cangc IS NOT FALSE;

	INSERT INTO
		t_del_fthumbs (fname,thumb)
	SELECT
		fname,thumb
	FROM
		(
			SELECT
				delfnt.fname,
				delfnt.thumb,
				allfnames.cangc
			FROM
				(
					SELECT DISTINCT
						fname,
						thumb
					FROM
						oldrows
				) AS delfnt
			LEFT JOIN
				LATERAL (
					SELECT
						(COUNT(*) == 0) AS cangc
					FROM
						ib0.files xf
					WHERE
						(delfnt.fname,delfnt.thumb) = (xf.fname,xf.thumb)
				) AS allfnt
			ON
				TRUE
		) AS gcft
	WHERE
		cangc IS NOT FALSE;

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
