package psqlib

// implements web imageboard interface v0

import (
	"../webib0"
)

func (sp PSQLProvider) IBGetBoardList(bl *ib0.IBBoardList) (error, int) {
	rows, err := sp.db.Query("SELECT bname,attrib FROM ib0.boards")
	if err != nil {
		return sqlError(err, "query"), http.StatusInternalServerError
	}
	var jcfg xtypes.JSONText
	bl.Boards = make([]ib0.IBBoardListBoard, 0)
	for rows.Next() {
		cfg := defaultBoardAttributes
		var b ib0.IBBoardListBoard
		err = rows.Scan(&b.Name, &jcfg)
		if err != nil {
			return sqlError(err, "rows scan"), http.StatusInternalServerError
		}
		err := jcfg.Unmarshal(&cfg)
		if err != nil {
			return sqlError(err, "json unmarshal"), http.StatusInternalServerError
		}
		b.Description = cfg.Description
		b.Tags = cfg.Tags
		bl.Boards = append(bl.Boards, b)
	}
	return nil, 0
}

func (sp PSQLProvider) IBGetThreadListPage(page *ib0.IBThreadListPage, board string, num uint32) (error, int) {
	var bid uint32
	var jcfg xtypes.JSONText
	err := sp.db.
		QueryRow("SELECT bid,attrib FROM ib0.boards WHERE bname=$1", board).
		Scan(&bid, &jcfg)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("board does not exist"), http.StatusNotFound
		}
		return sqlError(err, "row scan"), http.StatusInternalServerError
	}
	battrs := defaultBoardAttributes
	err = jcfg.Unmarshal(&battrs)
	if err != nil {
		return sqlError(err, "json unmarshal"), http.StatusInternalServerError
	}
	if battrs.PageLimit != 0 && num > battrs.PageLimit {
		return errors.New("page does not exist"), http.StatusNotFound
	}
	var allcount uint64
	err = sp.db.
		QueryRow("SELECT COUNT(*) FROM ib0.threads WHERE bid=$1", bid).
		Scan(&allcount)
	if err != nil {
		return sqlError(err, "row scan"), http.StatusInternalServerError
	}
	rows, err := sp.db.Query(
`SELECT tid,tname,attrib
FROM ib0.threads
WHERE bid=$1
ORDER BY bump DESC
LIMIT $2 OFFSET $3`,
		bid, battrs.ThreadsPerPage, num * battrs.ThreadsPerPage)
	if err != nil {
		return sqlError(err, "query"), http.StatusInternalServerError
	}
	var tids []uint64
	for rows.Next() {
		var t ib0.IBThreadListPageThread
		tattrib := defaultThreadAttributes
		var tid uint64
		err := rows.Scan(&tid, &t.ID, &jcfg)
		if err != nil {
			return sqlError(err, "rows scan"), http.StatusInternalServerError
		}
		err = jcfg.Unmarshal(&tattrib)
		if err != nil {
			return sqlError(err, "json unmarshal"), http.StatusInternalServerError
		}
		tids = append(tids,tid)
		page.Threads = append(page.Threads, t)
	}
	for i, tid := range tids {
		rows, err = sp.db.Query(
`SELECT pname,pid,author,trip,email,subject,pdate,message,attrib
FROM ib0.posts
WHERE bid=$1 AND tid=$2 AND pid=$2
UNION ALL
SELECT * FROM (
	SELECT * FROM (
		SELECT pname,pid,author,trip,email,subject,pdate,message,attrib
		FROM ib0.posts
		WHERE bid=$1 AND tid=$2 AND pid!=$2
		ORDER BY pdate DESC,pid DESC
		LIMIT 5
	) AS tt
	ORDER BY pdate ASC,pid ASC
) AS ttt`, bid, tid)
		if err != nil {
			return sqlError(err, "query"), http.StatusInternalServerError
		}
		for rows.Next() {
			var pi ib0.IBPostInfo1
			pattrib := defaultPostAttributes
			var pid uint64
			var pdate time.Time
			err = rows.Scan(&pi.ID,&pid,&pi.pname,&pi.trip,&pi.email,&pi.subject,&pdate,&pi.Message,&jcfg)
			if err != nil {
				return sqlError(err, "row scan"), http.StatusInternalServerError
			}
			err = jcfg.Unmarshal(&pattrib)
			if err != nil {
				return sqlError(err, "json unmarshal"), http.StatusInternalServerError
			}
			pi.Date = fmtTime(pdate)
			if tid != pid {
				page.Threads[i].Replies = append(page.Threads[i].Replies,pi)
			} else {
				page.Threads[i].OP = pi
			}
		}
	}
	
	*page = ib0.IBThreadListPage{
		Threads: []ib0.IBThreadListPageThread{{
			ID: "0123456789ABCDEF0123456789ABCDEF",
			OP: ib0.IBPostInfo1{
				ID:      "0123456789ABCDEF0123456789ABCDEF",
				Name:    "",
				Trip:    "",
				Subject: "test subject",
				Date:    "1980-09-11 14:30",
				Message: "test OP message",
				Files: []ib0.IBFileInfo{
					{
						ID:   "_test/1.png",
						Type: "image",
						Thumb: ib0.IBThumbInfo{
							ID:     "_test/1.png.jpg",
							Width:  128,
							Height: 128,
						},
						Original: "original test file.png",
						Options: map[string]interface{}{
							"width":  512,
							"height": 512,
						},
					},
				},
			},
			SkippedReplies: 0,
			Replies: []ib0.IBPostInfo1{
				{
					ID:      "00112233445566770011223344556677",
					Name:    "",
					Trip:    "",
					Subject: "",
					Date:    "1980-09-12 14:30",
					Message: "test reply message 1",
					Files:   []ib0.IBFileInfo{},
				},
				{
					ID:      "8899AABBCCDDEEFF8899AABBCCDDEEFF",
					Name:    "bob",
					Trip:    "",
					Subject: "",
					Date:    "1980-09-13 14:30",
					Message: "test reply message 2",
					Files: []ib0.IBFileInfo{
						{
							ID:   "_test/2.jpg",
							Type: "image",
							Thumb: ib0.IBThumbInfo{
								ID:     "_test/2.jpg.jpg",
								Width:  128,
								Height: 64,
							},
							Original: "original test file 2.jpg",
							Options: map[string]interface{}{
								"width":  512,
								"height": 256,
							},
						},
						{
							ID:   "_test/3.opus",
							Type: "audio",
							Thumb: ib0.IBThumbInfo{
								ID:     "_test/3.opus.jpg",
								Width:  128,
								Height: 128,
							},
							Original: "original test file 3.opus",
							Options:  map[string]interface{}{},
						},
					},
				},
			},
		}},
		Number:   num,
		Avaiable: 2,
	}
	return nil, 0
}

func (PSQLProvider) IBGetThreadCatalog(catalog *ib0.IBThreadCatalog, board string) (error, int) {
	if board != "test" {
		return errors.New("board does not exist"), http.StatusNotFound
	}
	*catalog = ib0.IBThreadCatalog{Threads: []ib0.IBThreadInfo1{
		{
			ID: "0123456789ABCDEF0123456789ABCDEF",
			Thumb: ib0.IBThumbInfo{
				ID:     "_test/1.png.jpg",
				Width:  128,
				Height: 128,
			},
			Subject: "test1",
			Message: "test message 1",
		},
		{
			ID: "00112233445566770011223344556677",
			Thumb: ib0.IBThumbInfo{
				ID:     "_test/2.jpg.jpg",
				Width:  128,
				Height: 64,
			},
			Subject: "test2",
			Message: "",
		},
		{
			ID: "8899AABBCCDDEEFF8899AABBCCDDEEFF",
			Thumb: ib0.IBThumbInfo{
				ID:     "_test/3.opus.jpg",
				Width:  128,
				Height: 128,
			},
			Subject: "",
			Message: "test message 3",
		},
	}}
	return nil, 0
}

func (PSQLProvider) IBGetThread(thread *ib0.IBThread, board string, threadid string) (error, int) {
	if board != "test" {
		return errors.New("board does not exist"), http.StatusNotFound
	}
	if threadid != "0123456789ABCDEF0123456789ABCDEF" {
		return errors.New("thread does not exist"), http.StatusNotFound
	}
	*thread = ib0.IBThread{
		ID: "0123456789ABCDEF0123456789ABCDEF",
		OP: ib0.IBPostInfo1{
			ID:      "0123456789ABCDEF0123456789ABCDEF",
			Name:    "",
			Trip:    "",
			Subject: "test subject",
			Date:    "1980-09-11 14:30",
			Message: "test OP message",
			Files: []ib0.IBFileInfo{
				{
					ID:   "_test/1.png",
					Type: "image",
					Thumb: ib0.IBThumbInfo{
						ID:     "_test/1.png.jpg",
						Width:  128,
						Height: 128,
					},
					Original: "original test file.png",
					Options: map[string]interface{}{
						"width":  512,
						"height": 512,
					},
				},
			},
		},
		Replies: []ib0.IBPostInfo1{
			{
				ID:      "00112233445566770011223344556677",
				Name:    "",
				Trip:    "",
				Subject: "",
				Date:    "1980-09-12 14:30",
				Message: "test reply message 1",
				Files:   []ib0.IBFileInfo{},
			},
			{
				ID:      "8899AABBCCDDEEFF8899AABBCCDDEEFF",
				Name:    "bob",
				Trip:    "",
				Subject: "",
				Date:    "1980-09-13 14:30",
				Message: "test reply message 2",
				Files: []ib0.IBFileInfo{
					{
						ID:   "_test/2.jpg",
						Type: "image",
						Thumb: ib0.IBThumbInfo{
							ID:     "_test/2.jpg.jpg",
							Width:  128,
							Height: 64,
						},
						Original: "original test file 2.jpg",
						Options: map[string]interface{}{
							"width":  512,
							"height": 256,
						},
					},
					{
						ID:   "_test/3.opus",
						Type: "audio",
						Thumb: ib0.IBThumbInfo{
							ID:     "_test/3.opus.jpg",
							Width:  128,
							Height: 128,
						},
						Original: "original test file 3.opus",
						Options:  map[string]interface{}{},
					},
				},
			},
		},
	}
	return nil, 0
}