package pireadweb

import (
	"net/http"

	xtypes "github.com/jmoiron/sqlx/types"

	"nksrv/lib/psqlib/internal/pibase"
	"nksrv/lib/psqlib/internal/pibaseweb"
	ib0 "nksrv/lib/webib0"
)

func GetBoardList(sp *pibase.PSQLIB, bl *ib0.IBBoardList) (error, int) {
	var err error

	rows, err := sp.StPrep[pibase.St_web_listboards].Query()
	if err != nil {
		return sp.SQLError("boards query", err), http.StatusInternalServerError
	}

	var jcfg xtypes.JSONText
	bl.Boards = make([]ib0.IBBoardListBoard, 0)

	for rows.Next() {
		var b ib0.IBBoardListBoard
		cfg := pibaseweb.DefaultBoardAttributes

		err = rows.Scan(&b.BNum, &b.Name, &b.Description, &jcfg, &b.NumThreads, &b.NumPosts)
		if err != nil {
			rows.Close()
			return sp.SQLError("boards query rows scan", err), http.StatusInternalServerError
		}

		err = jcfg.Unmarshal(&cfg)
		if err != nil {
			rows.Close()
			return sp.SQLError("board json unmarshal", err), http.StatusInternalServerError
		}

		b.Tags = cfg.Tags
		bl.Boards = append(bl.Boards, b)
	}
	if err = rows.Err(); err != nil {
		return sp.SQLError("boards query rows iteration", err), http.StatusInternalServerError
	}

	return nil, 0
}
