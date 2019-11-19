package psqlib

// implements web imageboard interface v0

import (
	"database/sql"
	"net/http"

	//. "nksrv/lib/logx"

	"nksrv/lib/ftypes"
	"nksrv/lib/mail"
	ib0 "nksrv/lib/webib0"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
)

// functionality

func (sp *PSQLIB) IBGetBoardList(bl *ib0.IBBoardList) (error, int) {
	var err error

	rows, err := sp.st_prep[st_web_listboards].Query()
	if err != nil {
		return sp.sqlError("boards query", err), http.StatusInternalServerError
	}

	var jcfg xtypes.JSONText
	bl.Boards = make([]ib0.IBBoardListBoard, 0)

	for rows.Next() {
		var b ib0.IBBoardListBoard
		cfg := defaultBoardAttributes

		err = rows.Scan(&b.BNum, &b.Name, &b.Description, &jcfg, &b.NumThreads, &b.NumPosts)
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

func (sp *PSQLIB) ensureThumb(
	t ib0.IBThumbInfo, fname, ftype string) ib0.IBThumbInfo {

	if t.ID == "" {
		t.Alt, t.Width, t.Height = sp.altthumb.GetAltThumb(fname, ftype)
	}
	return t
}

func webCleanHeaders(h mail.Headers) {
	delete(h, "Message-ID")
	delete(h, "MIME-Version")
	delete(h, "Content-Type")
}

func (sp *PSQLIB) IBGetThreadListPage(page *ib0.IBThreadListPage,
	board string, num uint32) (error, int) {

	rows, err := sp.st_prep[st_web_thread_list_page].Query(board, num)
	if err != nil {
		return sp.sqlError("Web_thread_list_page query", err),
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
			b_p_id         sql.NullInt64
			p_name         sql.NullString
			b_p_activ_refs xtypes.JSONText
			// xp
			msgid      sql.NullString
			pdate      pq.NullTime
			psage      sql.NullBool
			p_f_count  sql.NullInt64
			author     sql.NullString
			trip       sql.NullString
			title      sql.NullString
			message    []byte
			pheaders_j xtypes.JSONText
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

			&b_p_id, &p_name, &b_p_activ_refs,

			&msgid, &pdate, &psage, &p_f_count, &author, &trip, &title,
			&message, &pheaders_j,

			&f_id, &fname, &ftype, &fsize, &thumb, &oname,
			&filecfg_j, &thumbcfg_j)

		if err != nil {
			rows.Close()
			return sp.sqlError(
					"web_thread_list_page query rows scan", err),
				http.StatusInternalServerError
		}

		/*sp.log.LogPrintln(DEBUG, "sql thread list",
		bid, bdesc,
		t_id, t_name,
		b_p_id, p_name,
		f_id)*/

		if x_bid != bid {
			battrs := defaultBoardAttributes

			err = battrib_j.Unmarshal(&battrs)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"web_thread_list_page board attr json unmarshal", err),
					http.StatusInternalServerError
			}

			page.Board = ib0.IBBoardInfo{
				BNum:        bid,
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

			err = b_p_activ_refs.Unmarshal(&pi.References)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"web_thread_list_page post attr json unmarshal", err),
					http.StatusInternalServerError
			}

			err = pheaders_j.Unmarshal(&pi.Headers)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"web_thread_list_page post headers json unmarshal", err),
					http.StatusInternalServerError
			}
			if pi.Headers != nil {
				webCleanHeaders(pi.Headers)
			}

			pi.Num = uint64(b_p_id.Int64)
			pi.ID = p_name.String
			pi.MsgID = msgid.String
			pi.Subject = title.String
			pi.Name = author.String
			pi.Trip = trip.String
			//pi.Email =
			pi.Sage = psage.Bool
			pi.Date = pdate.Time.Unix()
			pi.Message = message

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
			if ft := ftypes.StringToFType(ftype.String); ft.Normal() {
				var fi ib0.IBFileInfo
				ta := defaultThumbAttributes

				err = thumbcfg_j.Unmarshal(&ta)
				if err != nil {
					rows.Close()
					return sp.sqlError(
							"web_thread_list_page thumbcfg json unmarshal", err),
						http.StatusInternalServerError
				}

				err = filecfg_j.Unmarshal(&fi.Options)
				if err != nil {
					rows.Close()
					return sp.sqlError(
							"web_thread_list_page filecfg json unmarshal", err),
						http.StatusInternalServerError
				}

				fi.ID = fname.String
				fi.Type = ftype.String
				fi.Thumb.ID = thumb.String
				fi.Thumb.Width = ta.Width
				fi.Thumb.Height = ta.Height
				fi.Original = oname.String
				fi.Size = fsize.Int64

				fi.Thumb = sp.ensureThumb(fi.Thumb, fi.ID, fi.Type)

				l_post.Files = append(l_post.Files, fi)

				x_fid = f_id.Int64
			}
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return sp.sqlError("web_thread_list_page query rows iteration", err),
			http.StatusInternalServerError
	}

	if x_bid == 0 {
		return errNoSuchBoard, http.StatusNotFound
	}
	if x_tid == 0 && num > 0 {
		return errNoSuchPage, http.StatusNotFound
	}

	return nil, 0
}

func (sp *PSQLIB) IBGetOverboardPage(page *ib0.IBOverboardPage, num uint32) (
	error, int) {

	page.Number = num
	page.Available = 10
	if page.Number >= page.Available {
		return errNoSuchPage, http.StatusNotFound
	}

	rows, err := sp.st_prep[st_web_overboard_page].Query(num, 10)
	if err != nil {
		return sp.sqlError("Web_overboard_page query", err),
			http.StatusInternalServerError
	}

	var x_bid boardID
	var x_tid postID
	var x_bpid postID
	var x_fid int64

	var l_thread *ib0.IBOverboardPageThread
	var l_post *ib0.IBPostInfo

	for rows.Next() {
		var (
			// xb
			bid   boardID
			bname string
			// xt
			t_id      postID
			t_name    string
			t_p_count sql.NullInt64
			t_f_count sql.NullInt64
			// xbp
			b_p_id         sql.NullInt64
			p_name         sql.NullString
			b_p_activ_refs xtypes.JSONText
			// xp
			msgid      sql.NullString
			pdate      pq.NullTime
			psage      sql.NullBool
			p_f_count  sql.NullInt64
			author     sql.NullString
			trip       sql.NullString
			title      sql.NullString
			message    []byte
			pheaders_j xtypes.JSONText
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
			&bid, &bname,

			&t_id, &t_name, &t_p_count, &t_f_count,

			&b_p_id, &p_name, &b_p_activ_refs,

			&msgid, &pdate, &psage, &p_f_count, &author, &trip, &title,
			&message, &pheaders_j,

			&f_id, &fname, &ftype, &fsize, &thumb, &oname,
			&filecfg_j, &thumbcfg_j)

		if err != nil {
			rows.Close()
			return sp.sqlError(
					"Web_overboard_page query rows scan", err),
				http.StatusInternalServerError
		}

		if x_bid != bid || x_tid != t_id {
			var t ib0.IBOverboardPageThread

			t.BNum = bid
			t.ID = t_name
			t.BoardName = bname
			if t_p_count.Int64 > 0 {
				// OP not included
				t.SkippedReplies = t_p_count.Int64 - 1
			}
			t.SkippedFiles = t_f_count.Int64

			page.Threads = append(page.Threads, t)

			l_thread = &page.Threads[len(page.Threads)-1]

			x_bid = bid
			x_tid = t_id
			x_bpid = 0
			x_fid = 0
		}

		if x_bpid != postID(b_p_id.Int64) {

			var pi ib0.IBPostInfo

			err = b_p_activ_refs.Unmarshal(&pi.References)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"Web_overboard_page post attr json unmarshal", err),
					http.StatusInternalServerError
			}

			err = pheaders_j.Unmarshal(&pi.Headers)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"Web_overboard_page post headers json unmarshal", err),
					http.StatusInternalServerError
			}
			if pi.Headers != nil {
				webCleanHeaders(pi.Headers)
			}

			pi.Num = uint64(b_p_id.Int64)
			pi.ID = p_name.String
			pi.MsgID = msgid.String
			pi.Subject = title.String
			pi.Name = author.String
			pi.Trip = trip.String
			//pi.Email =
			pi.Sage = psage.Bool
			pi.Date = pdate.Time.Unix()
			pi.Message = message

			if postID(b_p_id.Int64) == t_id {
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
			if ft := ftypes.StringToFType(ftype.String); ft.Normal() {
				var fi ib0.IBFileInfo
				ta := defaultThumbAttributes

				err = thumbcfg_j.Unmarshal(&ta)
				if err != nil {
					rows.Close()
					return sp.sqlError(
							"Web_overboard_page thumbcfg json unmarshal", err),
						http.StatusInternalServerError
				}

				err = filecfg_j.Unmarshal(&fi.Options)
				if err != nil {
					rows.Close()
					return sp.sqlError(
							"Web_overboard_page filecfg json unmarshal", err),
						http.StatusInternalServerError
				}

				fi.ID = fname.String
				fi.Type = ftype.String
				fi.Thumb.ID = thumb.String
				fi.Thumb.Width = ta.Width
				fi.Thumb.Height = ta.Height
				fi.Original = oname.String
				fi.Size = fsize.Int64

				fi.Thumb = sp.ensureThumb(fi.Thumb, fi.ID, fi.Type)

				l_post.Files = append(l_post.Files, fi)

				x_fid = f_id.Int64
			}
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return sp.sqlError("Web_overboard_page query rows iteration", err),
			http.StatusInternalServerError
	}

	if (x_bid == 0 || x_tid == 0) && num > 0 {
		return errNoSuchPage, http.StatusNotFound
	}

	return nil, 0
}

func (sp *PSQLIB) IBGetThreadCatalog(
	page *ib0.IBThreadCatalog, board string) (error, int) {

	rows, err := sp.st_prep[st_web_thread_catalog].Query(board)
	if err != nil {
		return sp.sqlError("Web_catalog query", err),
			http.StatusInternalServerError
	}

	var x_bid boardID
	var x_bpid postID

	for rows.Next() {
		var (
			// xb
			bid       boardID
			bdesc     string
			battrib_j xtypes.JSONText
			// xt
			t_id      sql.NullInt64
			t_name    sql.NullString
			t_p_count sql.NullInt64
			t_f_count sql.NullInt64
			t_bump    pq.NullTime
			// xbp
			b_p_id sql.NullInt64
			// xp
			pdate     pq.NullTime
			p_f_count sql.NullInt64
			author    sql.NullString
			trip      sql.NullString
			title     sql.NullString
			message   []byte
			// xf
			f_id       sql.NullInt64
			fname      sql.NullString
			ftype      sql.NullString
			thumb      sql.NullString
			thumbcfg_j xtypes.JSONText
		)

		err = rows.Scan(
			&bid, &bdesc, &battrib_j,

			&t_id, &t_name, &t_p_count, &t_f_count, &t_bump,

			&b_p_id,

			&pdate, &p_f_count, &author, &trip, &title, &message,

			&f_id, &fname, &ftype, &thumb, &thumbcfg_j)

		if err != nil {
			rows.Close()
			return sp.sqlError("Web_catalog query rows scan", err),
				http.StatusInternalServerError
		}

		if x_bid != bid {
			battrs := defaultBoardAttributes

			err = battrib_j.Unmarshal(&battrs)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"Web_catalog board attr json unmarshal", err),
					http.StatusInternalServerError
			}

			page.Board = ib0.IBBoardInfo{
				BNum:        bid,
				Name:        board,
				Description: bdesc,
				Info:        battrs.Info,
			}

			x_bid = bid
			x_bpid = 0
		}

		if x_bpid != postID(b_p_id.Int64) {

			var t ib0.IBThreadCatalogThread

			t.Num = uint64(b_p_id.Int64)
			t.ID = t_name.String
			if t_p_count.Int64 > 0 {
				// OP itself not included
				t.TotalReplies = t_p_count.Int64 - 1
			}
			// OP files not counted
			t.TotalFiles = t_f_count.Int64 - p_f_count.Int64
			t.BumpDate = t_bump.Time.Unix()
			t.Subject = title.String
			t.Message = message

			if f_id.Int64 != 0 {
				if ft := ftypes.StringToFType(ftype.String); ft.Normal() {
					ta := defaultThumbAttributes

					err = thumbcfg_j.Unmarshal(&ta)
					if err != nil {
						rows.Close()
						return sp.sqlError(
								"Web_catalog thumbcfg json unmarshal", err),
							http.StatusInternalServerError
					}

					t.Thumb.ID = thumb.String
					t.Thumb.Width = ta.Width
					t.Thumb.Height = ta.Height

					t.Thumb = sp.ensureThumb(t.Thumb, fname.String, ftype.String)

					goto thumbnailed
				}
			}
			// fallback if not found thumbnail above
			t.Thumb = sp.ensureThumb(t.Thumb, "", "")
		thumbnailed:
			// thumbnail done at this point

			page.Threads = append(page.Threads, t)

			x_bpid = postID(b_p_id.Int64)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return sp.sqlError(
				"Web_catalog query rows iteration", err),
			http.StatusInternalServerError
	}

	if x_bid == 0 {
		return errNoSuchBoard, http.StatusNotFound
	}

	return nil, 0
}

func (sp *PSQLIB) IBGetOverboardCatalog(
	page *ib0.IBOverboardCatalog) (error, int) {

	rows, err := sp.st_prep[st_web_overboard_catalog].Query(100)
	if err != nil {
		return sp.sqlError("web_overboard_catalog query", err),
			http.StatusInternalServerError
	}

	var x_bid boardID
	var x_bpid postID

	for rows.Next() {
		var (
			// xt
			bid       boardID
			bname     string
			t_id      sql.NullInt64
			t_name    sql.NullString
			t_p_count sql.NullInt64
			t_f_count sql.NullInt64
			t_bump    pq.NullTime
			// xbp
			b_p_id sql.NullInt64
			// xp
			pdate     pq.NullTime
			p_f_count sql.NullInt64
			author    sql.NullString
			trip      sql.NullString
			title     sql.NullString
			message   []byte
			// xf
			f_id       sql.NullInt64
			fname      sql.NullString
			ftype      sql.NullString
			thumb      sql.NullString
			thumbcfg_j xtypes.JSONText
		)

		err = rows.Scan(
			&bid, &bname, &t_id, &t_name,
			&t_p_count, &t_f_count, &t_bump,

			&b_p_id,

			&pdate, &p_f_count, &author, &trip, &title, &message,

			&f_id, &fname, &ftype, &thumb, &thumbcfg_j)

		if err != nil {
			rows.Close()
			return sp.sqlError(
					"web_overboard_catalog query rows scan", err),
				http.StatusInternalServerError
		}

		if x_bid != bid || x_bpid != postID(b_p_id.Int64) {

			var t ib0.IBOverboardCatalogThread

			t.BNum = bid
			t.BoardName = bname
			t.Num = uint64(b_p_id.Int64)
			t.ID = t_name.String
			if t_p_count.Int64 > 0 {
				// OP itself not included
				t.TotalReplies = t_p_count.Int64 - 1
			}
			// OP files not counted
			t.TotalFiles = t_f_count.Int64 - p_f_count.Int64
			t.BumpDate = t_bump.Time.Unix()
			t.Subject = title.String
			t.Message = message

			if f_id.Int64 != 0 {
				if ft := ftypes.StringToFType(ftype.String); ft.Normal() {
					ta := defaultThumbAttributes

					err = thumbcfg_j.Unmarshal(&ta)
					if err != nil {
						rows.Close()
						return sp.sqlError(
								"web_overboard_catalog thumbcfg json unmarshal", err),
							http.StatusInternalServerError
					}

					t.Thumb.ID = thumb.String
					t.Thumb.Width = ta.Width
					t.Thumb.Height = ta.Height

					t.Thumb = sp.ensureThumb(t.Thumb, fname.String, ftype.String)

					goto thumbnailed
				}
			}
			// fallback if not found thumbnail above
			t.Thumb = sp.ensureThumb(t.Thumb, "", "")
		thumbnailed:
			// thumbnail done at this point

			page.Threads = append(page.Threads, t)

			x_bid = bid
			x_bpid = postID(b_p_id.Int64)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return sp.sqlError(
				"web_overboard_catalog query rows iteration", err),
			http.StatusInternalServerError
	}

	return nil, 0
}

func (sp *PSQLIB) IBGetThread(page *ib0.IBThreadPage,
	board string, threadid string) (error, int) {

	rows, err := sp.st_prep[st_web_thread].Query(board, threadid)
	if err != nil {
		return sp.sqlError("Web_thread query", err),
			http.StatusInternalServerError
	}

	var x_bid boardID
	var x_tid postID
	var x_bpid postID
	var x_fid int64

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
			// xto
			t_pos sql.NullInt64
			// xbp
			b_p_id         sql.NullInt64
			p_name         sql.NullString
			b_p_activ_refs xtypes.JSONText
			// xp
			msgid      sql.NullString
			pdate      pq.NullTime
			psage      sql.NullBool
			p_f_count  sql.NullInt64
			author     sql.NullString
			trip       sql.NullString
			title      sql.NullString
			message    []byte
			pheaders_j xtypes.JSONText
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

			&t_pos,

			&b_p_id, &p_name, &b_p_activ_refs,

			&msgid, &pdate, &psage, &p_f_count, &author, &trip, &title,
			&message, &pheaders_j,

			&f_id, &fname, &ftype, &fsize, &thumb, &oname,
			&filecfg_j, &thumbcfg_j)

		if err != nil {
			rows.Close()
			return sp.sqlError(
					"web_thread query rows scan", err),
				http.StatusInternalServerError
		}

		if x_bid != bid {

			battrs := defaultBoardAttributes

			err = battrib_j.Unmarshal(&battrs)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"web_thread board attr json unmarshal", err),
					http.StatusInternalServerError
			}

			page.Board = ib0.IBBoardInfo{
				BNum:        bid,
				Name:        board,
				Description: bdesc,
				Info:        battrs.Info,
			}

			x_bid = bid
			x_tid = 0
			x_bpid = 0
			x_fid = 0
		}

		if x_tid != postID(t_id.Int64) {

			page.ID = t_name.String

			if t_p_count.Int64 > 0 {
				// OP not included
				page.ThreadStats.NumReplies = t_p_count.Int64 - 1
			}

			page.ThreadStats.NumFiles = t_f_count.Int64

			if threads_per_page > 0 {
				page.ThreadStats.PageNum = uint32(
					uint64(t_pos.Int64-1) / uint64(threads_per_page))
			} else {
				page.ThreadStats.PageNum = 0
			}

			x_tid = postID(t_id.Int64)
			x_bpid = 0
			x_fid = 0
		}

		if x_bpid != postID(b_p_id.Int64) {

			var pi ib0.IBPostInfo

			err = b_p_activ_refs.Unmarshal(&pi.References)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"web_thread post attr json unmarshal", err),
					http.StatusInternalServerError
			}

			err = pheaders_j.Unmarshal(&pi.Headers)
			if err != nil {
				rows.Close()
				return sp.sqlError(
						"web_thread post headers json unmarshal", err),
					http.StatusInternalServerError
			}
			if pi.Headers != nil {
				webCleanHeaders(pi.Headers)
			}

			pi.Num = uint64(b_p_id.Int64)
			pi.ID = p_name.String
			pi.MsgID = msgid.String
			pi.Subject = title.String
			pi.Name = author.String
			pi.Trip = trip.String
			//pi.Email =
			pi.Sage = psage.Bool
			pi.Date = pdate.Time.Unix()
			pi.Message = message

			if b_p_id.Int64 == t_id.Int64 {
				// OP
				page.OP = pi
				l_post = &page.OP
			} else {
				// reply
				page.Replies = append(page.Replies, pi)
				l_post = &page.Replies[len(page.Replies)-1]
			}

			x_bpid = postID(b_p_id.Int64)
			x_fid = 0
		}

		if x_fid != f_id.Int64 {
			if ft := ftypes.StringToFType(ftype.String); ft.Normal() {
				var fi ib0.IBFileInfo
				ta := defaultThumbAttributes

				err = thumbcfg_j.Unmarshal(&ta)
				if err != nil {
					rows.Close()
					return sp.sqlError(
							"web_thread thumbcfg json unmarshal", err),
						http.StatusInternalServerError
				}

				err = filecfg_j.Unmarshal(&fi.Options)
				if err != nil {
					rows.Close()
					return sp.sqlError(
							"web_thread filecfg json unmarshal", err),
						http.StatusInternalServerError
				}

				fi.ID = fname.String
				fi.Type = ftype.String
				fi.Thumb.ID = thumb.String
				fi.Thumb.Width = ta.Width
				fi.Thumb.Height = ta.Height
				fi.Original = oname.String
				fi.Size = fsize.Int64

				fi.Thumb = sp.ensureThumb(fi.Thumb, fi.ID, fi.Type)

				l_post.Files = append(l_post.Files, fi)

				x_fid = f_id.Int64
			}
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return sp.sqlError("web_thread query rows iteration", err),
			http.StatusInternalServerError
	}

	if x_bid == 0 {
		return errNoSuchBoard, http.StatusNotFound
	}
	if x_tid == 0 {
		return errNoSuchThread, http.StatusNotFound
	}

	return nil, 0
}
