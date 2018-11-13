package psqlib

// implements web imageboard interface v0

import (
	"database/sql"
	"net/http"
	"time"

	ib0 "nekochan/lib/webib0"

	xtypes "github.com/jmoiron/sqlx/types"
)

// functionality

// XXX this all stuff is horribly unoptimised and unatomic

func (sp *PSQLIB) IBGetBoardList(bl *ib0.IBBoardList) (error, int) {
	var err error

	rows, err := sp.db.DB.Query("SELECT bname,bdesc,attrib FROM ib0.boards")
	if err != nil {
		return sp.sqlError("boards query", err), http.StatusInternalServerError
	}

	var jcfg xtypes.JSONText
	bl.Boards = make([]ib0.IBBoardListBoard, 0)

	for rows.Next() {
		var b ib0.IBBoardListBoard
		cfg := defaultBoardAttributes

		err = rows.Scan(&b.Name, &b.Description, &jcfg)
		if err != nil {
			rows.Close()
			return sp.sqlError("boards query rows scan", err), http.StatusInternalServerError
		}

		err = jcfg.Unmarshal(&cfg)
		if err != nil {
			rows.Close()
			return sp.sqlError("board json unmarshal", err), http.StatusInternalServerError
		}

		b.Tags = cfg.Tags
		bl.Boards = append(bl.Boards, b)
	}
	if err = rows.Err(); err != nil {
		return sp.sqlError("boards query rows iteration", err), http.StatusInternalServerError
	}

	return nil, 0
}

/*
func (sp *PSQLIB) xxxIBGetThreadListPage(page *ib0.IBThreadListPage,
	board string, num uint32) (error, int) {

	var err error
	var bid boardID
	var jcfg, jcfg2 xtypes.JSONText

	st := `SELECT xb.bid,xb.attrib
FROM ib0.boards xb
LEFT JOIN ib0.threads xt USING (bid)
LEFT JOIN (

) AS xt USING (tid)`
	rows, err := sp.db.DB.Query(

}
*/

func (sp *PSQLIB) IBGetThreadListPage(page *ib0.IBThreadListPage,
	board string, num uint32) (error, int) {

	var err error
	var bid boardID
	var threadsPerPage, maxPages uint32
	var jcfg, jcfg2 xtypes.JSONText
	var bdesc string

	// XXX SQL needs more work

	err = sp.db.DB.
		QueryRow("SELECT bid,bdesc,attrib,threads_per_page,max_pages FROM ib0.boards WHERE bname=$1", board).
		Scan(&bid, &bdesc, &jcfg, &threadsPerPage, &maxPages)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoSuchBoard, http.StatusNotFound
		}
		return sp.sqlError("boards row query scan", err), http.StatusInternalServerError
	}

	battrs := defaultBoardAttributes
	err = jcfg.Unmarshal(&battrs)
	if err != nil {
		return sp.sqlError("board attr json unmarshal", err), http.StatusInternalServerError
	}

	if threadsPerPage <= 0 {
		threadsPerPage = 0
		maxPages = 1
	}
	if maxPages != 0 && num > maxPages {
		return errNoSuchPage, http.StatusNotFound
	}

	page.Board = ib0.IBBoardInfo{
		Name:        board,
		Description: bdesc,
		Info:        battrs.Info,
	}

	var allcount uint64
	err = sp.db.DB.
		QueryRow("SELECT COUNT(*) FROM ib0.threads WHERE bid=$1", bid).
		Scan(&allcount)
	if err != nil {
		return sp.sqlError("thread row count query scan", err), http.StatusInternalServerError
	}

	var pgslimit interface{}
	if threadsPerPage > 0 {
		pgslimit = threadsPerPage
	}
	rows, err := sp.db.DB.Query(
		`SELECT tid,tname,attrib
FROM ib0.threads
WHERE bid=$1
ORDER BY bump DESC,tid ASC
LIMIT $2 OFFSET $3`,
		bid, pgslimit, num*threadsPerPage)
	if err != nil {
		return sp.sqlError("threads query", err), http.StatusInternalServerError
	}

	page.Number = num
	if threadsPerPage > 0 {
		page.Available =
			uint32((allcount + uint64(threadsPerPage) - 1) /
				uint64(threadsPerPage))
	} else {
		page.Available = 1
	}

	type tpid struct {
		tid  postID
		pids []postID
	}
	var tpids []tpid

	for rows.Next() {
		var t ib0.IBThreadListPageThread
		tattrib := defaultThreadAttributes
		var tid postID

		err = rows.Scan(&tid, &t.ID, &jcfg)
		if err != nil {
			rows.Close()
			return sp.sqlError("threads query rows scan", err), http.StatusInternalServerError
		}

		err = jcfg.Unmarshal(&tattrib)
		if err != nil {
			rows.Close()
			return sp.sqlError("thread attrib json unmarshal", err), http.StatusInternalServerError
		}

		tpids = append(tpids, tpid{tid: tid, pids: []uint64{tid}})
		page.Threads = append(page.Threads, t)
	}
	if err = rows.Err(); err != nil {
		return sp.sqlError("threads query rows iteration", err), http.StatusInternalServerError
	}

	// one SQL query per thread, horrible
	for i := range tpids {
		tid := tpids[i].tid
		// OP, then 5 last posts, sorted ascending
		// TODO attachments
		rows, err = sp.db.DB.Query(
			`SELECT pname,pid,pdate,author,trip,title,message,attrib
FROM ib0.posts
WHERE bid=$1 AND pid=$2
UNION ALL
SELECT pname,pid,pdate,author,trip,title,message,attrib FROM (
	SELECT * FROM (
		SELECT pname,pid,pdate,padded,author,trip,title,message,attrib
		FROM ib0.posts
		WHERE bid=$1 AND tid=$2 AND pid!=$2
		ORDER BY padded DESC,pid DESC
		LIMIT 5
	) AS tt
	ORDER BY padded ASC,pid ASC
) AS ttt`, bid, tid)
		if err != nil {
			return sp.sqlError("posts query", err), http.StatusInternalServerError
		}

		for rows.Next() {
			var pi ib0.IBPostInfo
			pattrib := defaultPostAttributes
			var pid postID
			var pdate time.Time

			err = rows.Scan(&pi.ID, &pid, &pdate, &pi.Name, &pi.Trip, &pi.Subject, (*[]byte)(&pi.Message), &jcfg)
			if err != nil {
				rows.Close()
				return sp.sqlError("posts query rows scan", err), http.StatusInternalServerError
			}

			err = jcfg.Unmarshal(&pattrib)
			if err != nil {
				rows.Close()
				return sp.sqlError("post attrib json unmarshal", err), http.StatusInternalServerError
			}

			pi.Date = pdate.Unix()
			pi.References = pattrib.References

			if tid != pid {
				page.Threads[i].Replies = append(page.Threads[i].Replies, pi)
				tpids[i].pids = append(tpids[i].pids, pid)
			} else {
				page.Threads[i].OP = pi
			}
		}
		if err = rows.Err(); err != nil {
			return sp.sqlError("posts query rows iteration", err), http.StatusInternalServerError
		}
	}

	// one SQL query per post, outright bad
	for i := range tpids {
		for j, pid := range tpids[i].pids {
			var pi *ib0.IBPostInfo

			if j != 0 {
				pi = &page.Threads[i].Replies[j-1]
			} else {
				pi = &page.Threads[i].OP
			}

			rows, err := sp.db.DB.Query(
				`SELECT fname,ftype,fsize,thumb,oname,filecfg,thumbcfg
FROM ib0.files
WHERE bid=$1 AND pid=$2
ORDER BY fid ASC`,
				bid, pid)
			if err != nil {
				return sp.sqlError("files query", err), http.StatusInternalServerError
			}

			for rows.Next() {
				var fi ib0.IBFileInfo
				fattrib := make(map[string]interface{})
				tattrib := defaultThumbAttributes

				err = rows.Scan(&fi.ID, &fi.Type, &fi.Size, &fi.Thumb.ID, &fi.Original, &jcfg, &jcfg2)
				if err != nil {
					rows.Close()
					return sp.sqlError("files query rows scan", err), http.StatusInternalServerError
				}

				err = jcfg.Unmarshal(&fattrib)
				if err != nil {
					rows.Close()
					return sp.sqlError("file fattrib json unmarshal", err), http.StatusInternalServerError
				}

				err = jcfg2.Unmarshal(&tattrib)
				if err != nil {
					rows.Close()
					return sp.sqlError("file tattrib json unmarshal", err), http.StatusInternalServerError
				}

				fi.Options = fattrib
				fi.Thumb.Width, fi.Thumb.Height = tattrib.Width, tattrib.Height
				pi.Files = append(pi.Files, fi)
			}
			if err = rows.Err(); err != nil {
				return sp.sqlError("files query rows iteration", err), http.StatusInternalServerError
			}
		}
	}

	return nil, 0
}

func (sp *PSQLIB) IBGetThreadCatalog(page *ib0.IBThreadCatalog, board string) (error, int) {
	var err error
	var bid boardID
	var jcfg xtypes.JSONText
	var bdesc string

	// XXX SQL needs more work

	err = sp.db.DB.
		QueryRow("SELECT bid,bdesc,attrib FROM ib0.boards WHERE bname=$1", board).
		Scan(&bid, &bdesc, &jcfg)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoSuchBoard, http.StatusNotFound
		}
		return sp.sqlError("boards row query scan", err), http.StatusInternalServerError
	}

	battrs := defaultBoardAttributes
	err = jcfg.Unmarshal(&battrs)
	if err != nil {
		return sp.sqlError("board attr json unmarshal", err), http.StatusInternalServerError
	}

	page.Board = ib0.IBBoardInfo{
		Name:        board,
		Description: bdesc,
		Info:        battrs.Info,
	}

	rows, err := sp.db.DB.Query(
		`SELECT tid,tname,attrib,bump
FROM ib0.threads
WHERE bid=$1
ORDER BY bump DESC,tid ASC`,
		bid)
	if err != nil {
		return sp.sqlError("threads query", err), http.StatusInternalServerError
	}

	var tids []postID
	for rows.Next() {
		var t ib0.IBThreadCatalogThread
		tattrib := defaultThreadAttributes
		var tid postID
		var bdate time.Time

		err = rows.Scan(&tid, &t.ID, &jcfg, &bdate)
		if err != nil {
			rows.Close()
			return sp.sqlError("threads query rows scan", err), http.StatusInternalServerError
		}

		err = jcfg.Unmarshal(&tattrib)
		if err != nil {
			rows.Close()
			return sp.sqlError("thread attrib json unmarshal", err), http.StatusInternalServerError
		}

		t.BumpDate = bdate.Unix()

		tids = append(tids, tid)
		page.Threads = append(page.Threads, t)
	}
	if err = rows.Err(); err != nil {
		return sp.sqlError("threads query rows iteration", err), http.StatusInternalServerError
	}

	for i, tid := range tids {
		t := &page.Threads[i]
		// XXX dumb code xd
		err = sp.db.DB.
			QueryRow("SELECT title,message FROM ib0.posts WHERE bid=$1 AND pid=$2 LIMIT 1", bid, tid).
			Scan(&t.Subject, (*[]byte)(&t.Message))
		if err != nil {
			return sp.sqlError("posts row query scan", err), http.StatusInternalServerError
		}
		var fname string
		var ftype string
		err = sp.db.DB.
			QueryRow("SELECT fname,thumb,ftype,thumbcfg FROM ib0.files WHERE bid=$1 AND pid=$2 ORDER BY fid ASC LIMIT 1", bid, tid).
			Scan(&fname, &t.ID, &ftype, &jcfg)
		if err != nil {
			if err != sql.ErrNoRows {
				return sp.sqlError("files row query scan", err), http.StatusInternalServerError
			}

			t.Thumb.Alt, t.Thumb.Width, t.Thumb.Height = sp.altthumb.GetAltThumb("", "")
		} else {
			if t.ID == "" {
				t.Thumb.Alt, t.Thumb.Width, t.Thumb.Height = sp.altthumb.GetAltThumb(fname, ftype)
			} else {
				tattrib := defaultThumbAttributes

				err = jcfg.Unmarshal(&tattrib)
				if err != nil {
					return sp.sqlError("thumb attrib json unmarshal", err), http.StatusInternalServerError
				}

				t.Thumb.Width = tattrib.Width
				t.Thumb.Height = tattrib.Height
			}
		}
	}

	return nil, 0
}

func (sp *PSQLIB) IBGetThread(page *ib0.IBThreadPage,
	board string, threadid string) (error, int) {

	var err error
	var bid boardID
	var tid postID
	var jcfg, jcfg2 xtypes.JSONText
	var bdesc string

	// XXX SQL needs more work

	err = sp.db.DB.
		QueryRow("SELECT bid,bdesc,attrib FROM ib0.boards WHERE bname=$1 LIMIT 1", board).
		Scan(&bid, &bdesc, &jcfg)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoSuchBoard, http.StatusNotFound
		}
		return sp.sqlError("boards row query scan", err), http.StatusInternalServerError
	}

	battrs := defaultBoardAttributes
	err = jcfg.Unmarshal(&battrs)
	if err != nil {
		return sp.sqlError("board attr json unmarshal", err), http.StatusInternalServerError
	}

	page.Board = ib0.IBBoardInfo{
		Name:        board,
		Description: bdesc,
		Info:        battrs.Info,
	}

	err = sp.db.DB.QueryRow(`SELECT tid,attrib FROM ib0.threads WHERE bid=$1 AND tname=$2 LIMIT 1`,
		bid, threadid).
		Scan(&tid, &jcfg)
	if err != nil {
		if err == sql.ErrNoRows {
			return errNoSuchThread, http.StatusNotFound
		}
		return sp.sqlError("thread query scan", err), http.StatusInternalServerError
	}

	page.ID = threadid

	tattrs := defaultThreadAttributes
	err = jcfg.Unmarshal(&tattrs)
	if err != nil {
		return sp.sqlError("thread attr json unmarshal", err), http.StatusInternalServerError
	}

	rows, err := sp.db.DB.Query(
		`SELECT pname,pid,author,trip,title,pdate,message,attrib
FROM ib0.posts
WHERE bid=$1 AND tid=$2
ORDER BY padded ASC,pid ASC`,
		bid, tid)
	if err != nil {
		return sp.sqlError("posts query", err), http.StatusInternalServerError
	}

	pids := []postID{tid}

	for rows.Next() {
		var pi ib0.IBPostInfo
		pattrib := defaultPostAttributes
		var pid postID
		var pdate time.Time

		err = rows.Scan(&pi.ID, &pid, &pi.Name, &pi.Trip, &pi.Subject, &pdate, (*[]byte)(&pi.Message), &jcfg)
		if err != nil {
			rows.Close()
			return sp.sqlError("posts query rows scan", err), http.StatusInternalServerError
		}

		err = jcfg.Unmarshal(&pattrib)
		if err != nil {
			rows.Close()
			return sp.sqlError("post attrib json unmarshal", err), http.StatusInternalServerError
		}

		pi.Date = pdate.Unix()
		pi.References = pattrib.References

		if tid != pid {
			page.Replies = append(page.Replies, pi)
			pids = append(pids, pid)
		} else {
			page.OP = pi
		}
	}
	if err = rows.Err(); err != nil {
		return sp.sqlError("posts query rows iteration", err), http.StatusInternalServerError
	}

	for i, pid := range pids {
		var pi *ib0.IBPostInfo

		if i != 0 {
			pi = &page.Replies[i-1]
		} else {
			pi = &page.OP
		}

		// one query per post, outright bad
		rows, err := sp.db.DB.Query(
			`SELECT fname,ftype,fsize,thumb,oname,filecfg,thumbcfg
FROM ib0.files
WHERE bid=$1 AND pid=$2
ORDER BY fid ASC`,
			bid, pid)
		if err != nil {
			return sp.sqlError("files query", err), http.StatusInternalServerError
		}

		for rows.Next() {
			var fi ib0.IBFileInfo
			fattrib := make(map[string]interface{})
			tattrib := defaultThumbAttributes

			err = rows.Scan(&fi.ID, &fi.Type, &fi.Size, &fi.Thumb.ID, &fi.Original, &jcfg, &jcfg2)
			if err != nil {
				rows.Close()
				return sp.sqlError("files query rows scan", err), http.StatusInternalServerError
			}

			err = jcfg.Unmarshal(&fattrib)
			if err != nil {
				rows.Close()
				return sp.sqlError("file fattrib json unmarshal", err), http.StatusInternalServerError
			}

			err = jcfg2.Unmarshal(&tattrib)
			if err != nil {
				rows.Close()
				return sp.sqlError("file tattrib json unmarshal", err), http.StatusInternalServerError
			}

			fi.Options = fattrib
			fi.Thumb.Width, fi.Thumb.Height = tattrib.Width, tattrib.Height
			pi.Files = append(pi.Files, fi)
		}
		if err = rows.Err(); err != nil {
			return sp.sqlError("files query rows iteration", err), http.StatusInternalServerError
		}
	}

	return nil, 0
}
