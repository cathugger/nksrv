package pipostinsert

import "github.com/lib/pq"

func sqlerrIsDuplicate(err error) bool {
	pqerr, ok := err.(*pq.Error)
	return ok && pqerr.Code == "23505"
}
