package psqlib

import (
	"fmt"
	"io"
	"time"
)

func (sp *PSQLIB) nntpGenerate(w io.Writer, num uint64, msgid CoreMsgIDStr) error {
	// TODO
	// need to generate article off database shit
	// im too fucking lazy to actually do it atm
	// so placeholder to test shit will work for now

	// fetch info about post. some of info we don't care about
	q := `SELECT jp.title,jp.message,jp.headers,jp.layout,jf.fname
FROM ib0.posts AS jp
JOIN ib0.files AS jf
USING (bid,pid)
WHERE jp.msgid = $1
ORDER BY jf.fid`
	rows, err := sp.db.DB.Query(q, string(msgid))
	if err != nil {
		return sp.sqlError("posts x files query", err)
	}

	pi := postInfo{}

	for rows.Next() {
		var fi fileInfo

		// XXX is it okay to overwrite stuff there?
		err = rows.Scan(&pi.MI.Title, &pi.MI.Message, &pi.H, &pi.L, &fi.ID)
		if err != nil {
			rows.Close()
			return sp.sqlError("posts x files query rows scan", err)
		}

		if fi.ID != "" {
			pi.FI = append(pi.FI, fi)
		}
	}
	if err = rows.Err(); err != nil {
		return sp.sqlError("posts x files query rows iteration", err)
	}

	// ensure Message-ID
	if len(pi.H["Message-ID"]) == 0 {
		pi.H["Message-ID"] = []string{fmt.Sprintf("<%s>", msgid)}
	}

	// ensure Subject
	if len(pi.H["Subject"]) == 0 && pi.MI.Title != "" {
		pi.H["Subject"] = []string{pi.MI.Title}
	}

	// TODO
	fmt.Fprintf(w, "Message-ID: <%s>\n\n", string(msgid))
	for i := 0; i < 20; i++ {
		time.Sleep(250 * time.Millisecond)
		fmt.Fprintf(w, "faggot\n")
	}
	return nil
}
