package pimod

import (
	"database/sql"
	"io"
	"os"
	"strings"
	"time"

	"nksrv/lib/app/mailib"
	. "nksrv/lib/utils/logx"
	mm "nksrv/lib/utils/minimail"
	"nksrv/lib/utils/text/bufreader"

	"nksrv/lib/app/psqlib/internal/pibase"
	"nksrv/lib/app/psqlib/internal/pibasemod"
	"nksrv/lib/app/psqlib/internal/pibasenntp"
)

func ModCmdDelete(
	mc *modCtx,
	gpid pibase.TPostID, bid pibase.TBoardID, bpid pibase.TPostID,
	pi mailib.PostInfo,
	selfid, ref pibasenntp.TCoreMsgIDStr,
	cmd string, args []string,
) (
	err error,
) {

	if len(args) == 0 {
		return
	}

	fmsgids := TFullMsgIDStr(args[0])
	if !mm.ValidMessageIDStr(fmsgids) {
		return
	}
	cmsgids := cutMsgID(fmsgids)
	if cmsgids == selfid || cmsgids == ref {
		return
	}

	err = BanByMsgID(sp, tx, cmsgids, bid, bpid, pi.MI.Title)
	if err != nil {
		return
	}

	return
}

func getModCmdInput(
	pi mailib.PostInfo, filenames []string) (io.Reader, io.Closer, error) {

	if pi.E.TextAttachment <= 0 {
		return strings.NewReader(pi.MI.Message), nil, nil
	}
	f, err := os.Open(filenames[pi.E.TextAttachment-1])
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}

func ExecModCmd(
	mc *modCtx,
	gpid pibase.TPostID, bid pibase.TBoardID, bpid pibase.TPostID,
	modid uint64, modCC pibasemod.ModCombinedCaps,
	pi mailib.PostInfo, filenames []string,
	selfid, ref pibasenntp.TCoreMsgIDStr,
) (
	err error, inputerr bool,
) {

	r, c, err := getModCmdInput(pi, filenames)
	if err != nil {
		inputerr = true
		return
	}
	if c != nil {
		defer c.Close()
	}

	var linebuf [2048]byte
	br := bufreader.NewBufReaderSize(r, 1024)
	for {
		var read int
		read, err = br.ReadUntil(linebuf[:], '\n')
		if err != nil && err != io.EOF {
			if err == bufreader.ErrDelimNotFound {
				// skip dis line it's too long
				// XXX maybe log warning
				// drain
				for {
					_, err = br.ReadUntil(linebuf[:], '\n')
					if err != bufreader.ErrDelimNotFound {
						break
					}
				}
				continue
			}
			// an actual error while reading
			return
		}

		hadeof := err == io.EOF
		err = nil

		unsafe_line := unsafeBytesToStr(linebuf[:read])
		unsafe_fields := strings.Fields(unsafe_line)

		if len(unsafe_fields) != 0 {

			unsafe_cmd := strings.ToLower(unsafe_fields[0])
			unsafe_args := unsafe_fields[1:]

			// TODO log commands we couldn't understand
			switch unsafe_cmd {
			case "delete":
				// TODO per-board stuff
				// TODO TODO TODO
				if modCC.ModCap.Cap&pibasemod.Cap_DelPost != 0 {
					// global delete by msgid
					err =
						ModCmdDelete(
							mc,
							gpid, bid, bpid, pi, selfid, ref,
							unsafe_cmd, unsafe_args)
			}
			if err != nil {
				return
			}
		}

		// EOF
		if hadeof {
			break
		}
	}

	err = nil
	return
}

func xxxx(
	sp *pibase.PSQLIB,
	tx *sql.Tx,
	modid uint64, modCC pibasemod.ModCombinedCaps) (
	err error) {

	srcdir := sp.Src.Main()
	xst := tx.Stmt(sp.StPrep[pibase.St_mod_fetch_and_clear_mod_msgs_continue])

	// 666 days in the future
	off_pdate := time.Now().Add(time.Hour * 24 * 666).UTC()
	off_g_p_id := uint64(0)
	off_b_id := uint32(0)

	type idt struct {
		bid  pibase.TBoardID
		bpid pibase.TPostID
	}
	type postinfo struct {
		gpid      pibase.TPostID
		xid       idt
		bname     string
		msgid     string
		ref       string
		title     string
		date_sent time.Time
		message   string
		txtidx    uint32
		files     []string
	}
	var posts []postinfo
	lastx := idt{0, 0}

requery:
	for {
		var rows *sql.Rows
		rows, err = xst.Query(modid, off_pdate, off_g_p_id, off_b_id)
		if err != nil {
			err = sp.SQLError("st_web_fetch_and_clear_mod_msgs query", err)
			return
		}

		for rows.Next() {
			/*
				zbp.date_sent,
				zbp.g_p_id,
				zbp.b_id,
				zbp.b_p_id,

				yb.b_name,
				yp.msgid,
				ypbp.msgid,

				yp.title,
				yp.message,
				yp.extras -> 'text_attach',
				yf.fname
			*/
			var p postinfo
			var ref, fname sql.NullString
			var txtidx sql.NullInt64

			err = rows.Scan(
				&p.date_sent, &p.gpid, &p.xid.bid, &p.xid.bpid,
				&p.bname, &p.msgid, &ref,
				&p.title, &p.message, &txtidx, &fname)
			if err != nil {
				rows.Close()
				err = sp.SQLError("st_web_fetch_and_clear_mod_msgs rows scan", err)
				return
			}

			if lastx != p.xid {
				lastx = p.xid
				p.ref = ref.String
				p.txtidx = uint32(txtidx.Int64)
				p.date_sent = p.date_sent.UTC()
				posts = append(posts, p)
			}
			if fname.String != "" {
				pp := &posts[len(posts)-1]
				pp.files = append(pp.files, srcdir+fname.String)
			}
		}
		if err = rows.Err(); err != nil {
			err = sp.SQLError("st_web_fetch_and_clear_mod_msgs rows it", err)
			return
		}

		for i := range posts {
			// prepare postinfo good enough for execModCmd
			pi := mailib.PostInfo{
				MessageID: pibasenntp.TCoreMsgIDStr(posts[i].msgid),
				Date:      posts[i].date_sent,
				MI: mailib.MessageInfo{
					Title:   posts[i].title,
					Message: posts[i].message,
				},
				E: mailib.PostExtraAttribs{
					TextAttachment: posts[i].txtidx,
				},
			}

			sp.Log.LogPrintf(DEBUG,
				"setmodpriv: executing <%s> from board[%s]",
				posts[i].msgid, posts[i].bname)

			var inputerr bool

			del_our_mod_id, err, inputerr = ExecModCmd(
				sp,
				tx, posts[i].gpid, posts[i].xid.bid, posts[i].xid.bpid,
				modid, modCC,
				pi, posts[i].files, pi.MessageID,
				pibasenntp.TCoreMsgIDStr(posts[i].ref))

			if err != nil {

				if inputerr {
					// mod msg is just fucked at this point
					// this shouldn't happen
					// XXX should we delete this bad msg??
					sp.Log.LogPrintf(ERROR,
						"setmodpriv: [proceeding anyway] inputerr while execing <%s>: %v",
						posts[i].msgid, err)
					err = nil
					continue
				}

				// return err directly
				return
			}

			if del_our_mod_id {
				sp.Log.LogPrintf(
					DEBUG, "setmodpriv: delmodid %d is ours, requerying", modid)
				// msg we just deleted was made by mod we just upp'd
				// that means that it may be msg in query we just made
				// it's unsafe to proceed with current cached query
				off_pdate = posts[i].date_sent
				off_g_p_id = posts[i].gpid
				off_b_id = posts[i].xid.bid
				posts = posts[:0]
				continue requery
			}
		}

		if len(posts) < 4096 {
			// if less than limit that means we dont need another query
			break
		} else {
			// issue another query, there may be more data
			i := len(posts) - 1
			off_pdate = posts[i].date_sent
			off_g_p_id = posts[i].gpid
			off_b_id = posts[i].xid.bid
			posts = posts[:0]
			continue requery
		}
	}

	return
}
