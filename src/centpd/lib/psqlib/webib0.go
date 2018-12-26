package psqlib

// implements web imageboard interface v0

import (
	"database/sql"
	"net/http"
	"time"

	ib0 "centpd/lib/webib0"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
)

// functionality

// XXX this all stuff is horribly unoptimised and unatomic

func (sp *PSQLIB) IBGetBoardList(bl *ib0.IBBoardList) (error, int) {
	var err error

	q := st_list[st_Web_listboards]
	rows, err := sp.db.DB.Query(q)
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

func (sp *PSQLIB) ensureThumb(
	t ib0.IBThumbInfo, fname, ftype string) ib0.IBThumbInfo {

	if t.ID == "" {
		t.Alt, t.Width, t.Height = sp.altthumb.GetAltThumb(fname, ftype)
	}
	return t
}

func (sp *PSQLIB) IBGetThreadListPage(page *ib0.IBThreadListPage,
	board string, num uint32) (error, int) {

	q := st_list[st_Web_thread_list_page]

	rows, err := sp.db.DB.Query(q, board, num)
	if err != nil {
		return sp.sqlError("BxTxBPxPxF query", err),
			http.StatusInternalServerError
	}

	var x_bid boardID
	var x_tid postID
	var x_bpid postID
	var x_fid int64

	var l_thread *ib0.IBThreadListPageThread
	var l_post *ib0.IBPostInfo

	for rows.Next() {
		var (
			// xb
			bid              boardID
			bdesc            string
			battrib_j        xtypes.JSONText
			threads_per_page uint32
			t_count          uint64
			// xt
			t_id      sql.NullInt64
			t_name    sql.NullString
			t_p_count sql.NullInt64
			t_f_count sql.NullInt64
			// xbp
			b_p_id sql.NullInt64
			p_name sql.NullString
			// xp
			pdate     pq.NullTime
			psage     sql.NullBool
			p_f_count sql.NullInt64
			author    sql.NullString
			trip      sql.NullString
			title     sql.NullString
			message   []byte
			pattrib_j xtypes.JSONText
			// xf
			f_id       sql.NullInt64
			fname      sql.NullString
			ftype      sql.NullString
			fsize      sql.NullInt64
			thumb      sql.NullString
			oname      sql.NullString
			filecfg_j  xtypes.JSONText
			thumbcfg_j xtypes.JSONText
		)

		err = rows.Scan(
			&bid, &bdesc, &battrib_j, &threads_per_page, &t_count,

			&t_id, &t_name, &t_p_count, &t_f_count,

			&b_p_id, &p_name,

			&pdate, &psage, &p_f_count, &author, &trip, &title,
			&message, &pattrib_j,

			&f_id, &fname, &ftype, &fsize, &thumb, &oname,
			&filecfg_j, &thumbcfg_j)
		if err != nil {
			rows.Close()
			return sp.sqlError("BxTxBPxPxF query rows scan", err), http.StatusInternalServerError
		}

		if x_bid != bid {
			battrs := defaultBoardAttributes

			err = battrib_j.Unmarshal(&battrs)
			if err != nil {
				rows.Close()
				return sp.sqlError("board attr json unmarshal", err),
					http.StatusInternalServerError
			}

			page.Board = ib0.IBBoardInfo{
				Name:        board,
				Description: bdesc,
				Info:        battrs.Info,
			}
			page.Number = num
			if threads_per_page > 0 {
				page.Available = uint32(
					(t_count + uint64(threads_per_page) - 1) /
						uint64(threads_per_page))
			}
			if page.Available <= 0 {
				page.Available = 1
			}

			x_bid = bid
			x_tid = 0
			x_bpid = 0
			x_fid = 0
		}

		if x_tid != postID(t_id.Int64) {
			var t ib0.IBThreadListPageThread

			t.ID = t_name.String
			if t_p_count.Int64 > 0 {
				// OP not included
				t.SkippedReplies = t_p_count.Int64 - 1
			}
			t.SkippedFiles = t_f_count.Int64

			page.Threads = append(page.Threads, t)

			l_thread = &page.Threads[len(page.Threads)-1]

			x_tid = postID(t_id.Int64)
			x_bpid = 0
			x_fid = 0
		}

		if x_bpid != postID(b_p_id.Int64) {
			var pi ib0.IBPostInfo
			pattrib := defaultPostAttributes

			err = pattrib_j.Unmarshal(&pattrib)
			if err != nil {
				rows.Close()
				return sp.sqlError("post attr json unmarshal", err),
					http.StatusInternalServerError
			}

			pi.ID = p_name.String
			pi.Subject = title.String
			pi.Name = author.String
			pi.Trip = trip.String
			//pi.Email =
			pi.Sage = psage.Bool
			pi.Date = pdate.Time.Unix()
			pi.Message = message
			pi.References = pattrib.References

			if b_p_id.Int64 == t_id.Int64 {
				// OP

				l_thread.OP = pi

				l_post = &l_thread.OP
			} else {
				// reply

				l_thread.SkippedReplies--

				l_thread.Replies = append(l_thread.Replies, pi)

				l_post = &l_thread.Replies[len(l_thread.Replies)-1]
			}

			l_thread.SkippedFiles -= p_f_count.Int64

			x_bpid = postID(b_p_id.Int64)
			x_fid = 0
		}

		if x_fid != f_id.Int64 {
			var fi ib0.IBFileInfo
			ta := defaultThumbAttributes

			err = thumbcfg_j.Unmarshal(&ta)
			if err != nil {
				rows.Close()
				return sp.sqlError("thumbcfg json unmarshal", err),
					http.StatusInternalServerError
			}

			err = filecfg_j.Unmarshal(&fi.Options)
			if err != nil {
				rows.Close()
				return sp.sqlError("filecfg json unmarshal", err),
					http.StatusInternalServerError
			}

			fi.ID = fname.String
			fi.Type = ftype.String
			fi.Thumb.ID = thumb.String
			fi.Thumb.Width = ta.Width
			fi.Thumb.Height = ta.Height
			fi.Original = oname.String
			fi.Size = fsize.Int64

			l_post.Files = append(l_post.Files, fi)

			x_fid = f_id.Int64
		}
	}

	if x_bid == 0 {
		return errNoSuchBoard, http.StatusNotFound
	}
	if x_tid == 0 && num > 0 {
		return errNoSuchPage, http.StatusNotFound
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
			Scan(&fname, &t.Thumb.ID, &ftype, &jcfg)
		if err != nil {
			if err != sql.ErrNoRows {
				return sp.sqlError("files row query scan", err), http.StatusInternalServerError
			}

			t.Thumb.Alt, t.Thumb.Width, t.Thumb.Height = sp.altthumb.GetAltThumb("", "")
		} else {
			if t.Thumb.ID == "" {
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

		err = rows.Scan(
			&pi.ID, &pid, &pi.Name, &pi.Trip, &pi.Subject, &pdate,
			(*[]byte)(&pi.Message), &jcfg)
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

			err = rows.Scan(
				&fi.ID, &fi.Type, &fi.Size, &fi.Thumb.ID, &fi.Original,
				&jcfg, &jcfg2)
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
			fi.Thumb = sp.ensureThumb(fi.Thumb, fi.ID, fi.Type)
			pi.Files = append(pi.Files, fi)
		}
		if err = rows.Err(); err != nil {
			return sp.sqlError("files query rows iteration", err), http.StatusInternalServerError
		}
	}

	return nil, 0
}
