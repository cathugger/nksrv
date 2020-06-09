package pireadweb

import (
	"database/sql"
	"net/http"

	xtypes "github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"

	"nksrv/lib/ftypes"
)

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
