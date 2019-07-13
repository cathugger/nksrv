package psqlib

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	//. "nksrv/lib/logx"
	"nksrv/lib/fstore"
	"nksrv/lib/mail"
	"nksrv/lib/mailib"

	xtypes "github.com/jmoiron/sqlx/types"
)

type NormalFileList struct {
	FI  []mailib.FileInfo
	src *fstore.FStore
}

func (fl NormalFileList) OpenFileAt(i int) (io.ReadCloser, error) {
	return os.Open(fl.src.Main() + fl.FI[i].ID)
}

func (sp *PSQLIB) nntpGenerate(
	w io.Writer, msgid CoreMsgIDStr, gpid postID) (err error) {

	// fetch info about post. some of info we don't care about
	rows, err := sp.st_prep[st_NNTP_articleGetByGPID].Query(gpid)
	if err != nil {
		return sp.sqlError("posts x files query", err)
	}

	pi := mailib.PostInfo{}

	havesomething := false

	for rows.Next() {
		var jH, jL xtypes.JSONText
		var fid sql.NullString
		var fsize sql.NullInt64

		// XXX is it okay to overwrite stuff there?
		err = rows.Scan(
			&pi.MI.Title, &pi.MI.Message, &jH, &jL,
			&fid, &fsize)
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
			pi.FI = append(pi.FI, mailib.FileInfo{
				ID:   fid.String,
				Size: fsize.Int64,
			})
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

	return mailib.GenerateMessage(w, pi, NormalFileList{FI: pi.FI, src: &sp.src})
}
