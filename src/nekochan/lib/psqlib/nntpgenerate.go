package psqlib

import (
	"database/sql"
	"fmt"
	"io"

	. "nekochan/lib/logx"
	"nekochan/lib/mail"
	"nekochan/lib/mailib"

	xtypes "github.com/jmoiron/sqlx/types"
)

func (sp *PSQLIB) nntpGenerate(
	w io.Writer, num uint64, msgid CoreMsgIDStr) (err error) {

	// fetch info about post. some of info we don't care about
	q := `SELECT jp.title,jp.message,jp.headers,jp.layout,jf.fname
FROM ib0.posts AS jp
LEFT JOIN ib0.files AS jf
USING (bid,pid)
WHERE jp.msgid = $1
ORDER BY jf.fid`
	rows, err := sp.db.DB.Query(q, string(msgid))
	if err != nil {
		return sp.sqlError("posts x files query", err)
	}

	pi := mailib.PostInfo{}

	havesomething := false

	for rows.Next() {
		var jH, jL xtypes.JSONText
		var fid sql.NullString

		// XXX is it okay to overwrite stuff there?
		err = rows.Scan(&pi.MI.Title, &pi.MI.Message, &jH, &jL, &fid)
		if err != nil {
			rows.Close()
			return sp.sqlError("posts x files query rows scan", err)
		}

		//sp.log.LogPrintf(DEBUG,
		//	"nntpGenerate: PxF: title(%q) msg(%q) H(%q) L(%q) id(%v)",
		//	pi.MI.Title, pi.MI.Message, jH, jL, fid)

		if !havesomething {
			err = jH.Unmarshal(&pi.H)
			if err != nil {
				rows.Close()
				return sp.sqlError("jH unmarshal", err)
			}

			err = jL.Unmarshal(&pi.L)
			if err != nil {
				rows.Close()
				return sp.sqlError("jL unmarshal", err)
			}

			//sp.log.LogPrintf(DEBUG,
			//	"nntpGenerate: unmarshaled H(%#v) L(%#v)",
			//	pi.H, &pi.L)
		}

		if fid.Valid && fid.String != "" {
			pi.FI = append(pi.FI, mailib.FileInfo{ID: fid.String})
		}

		havesomething = true
	}
	if err = rows.Err(); err != nil {
		return sp.sqlError("posts x files query rows iteration", err)
	}

	if !havesomething {
		return errNotExist
	}

	// ensure Message-ID
	if len(pi.H["Message-ID"]) == 0 {
		pi.H["Message-ID"] = mail.OneHeaderVal(fmt.Sprintf("<%s>", msgid))
	}

	// ensure Subject
	if len(pi.H["Subject"]) == 0 && pi.MI.Title != "" {
		pi.H["Subject"] = mail.OneHeaderVal(pi.MI.Title)
	}

	return mailib.GenerateMessage(&sp.src, w, pi)
}
