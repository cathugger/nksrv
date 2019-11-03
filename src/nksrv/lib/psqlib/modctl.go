package psqlib

import (
	"database/sql"
	"io"
	"os"
	"strings"
	"time"

	"nksrv/lib/bufreader"
	. "nksrv/lib/logx"
	"nksrv/lib/mailib"
	mm "nksrv/lib/minimail"
)

func (sp *PSQLIB) modCmdDelete(
	tx *sql.Tx, gpid postID, bid boardID, bpid postID,
	pi mailib.PostInfo, selfid, ref CoreMsgIDStr,
	cmd string, args []string,
	in_delmsgids delMsgIDState, in_delmodids delModIDState) (
	out_delmsgids delMsgIDState, out_delmodids delModIDState,
	err error) {

	if len(args) == 0 {
		return
	}

	fmsgids := FullMsgIDStr(args[0])
	if !mm.ValidMessageIDStr(fmsgids) {
		return
	}
	cmsgids := cutMsgID(fmsgids)
	if cmsgids == selfid || cmsgids == ref {
		return
	}

	out_delmsgids, out_delmodids, err =
		sp.banByMsgID(
			tx, cmsgids, bid, bpid, pi.MI.Title,
			in_delmsgids, in_delmodids)
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

func (sp *PSQLIB) execModCmd(
	tx *sql.Tx, gpid postID, bid boardID, bpid postID,
	modid uint64, modCC ModCombinedCaps,
	pi mailib.PostInfo, filenames []string,
	selfid, ref CoreMsgIDStr,
	_in_delmsgids delMsgIDState, _in_delmodids delModIDState) (
	out_delmsgids delMsgIDState, out_delmodids delModIDState,
	err error, inputerr bool) {

	out_delmsgids = _in_delmsgids
	out_delmodids = _in_delmodids

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

			cmd := strings.ToLower(unsafe_fields[0])
			args := unsafe_fields[1:]

			// TODO log commands we couldn't understand
			switch cmd {
			case "delete":
				// TODO per-board stuff
				// TODO TODO TODO
				if modCC.ModCap.Cap&cap_delpost != 0 {
					// global delete by msgid
					out_delmsgids, out_delmodids, err =
						sp.modCmdDelete(
							tx, gpid, bid, bpid, pi, selfid, ref, cmd, args,
							out_delmsgids, out_delmodids)
				}
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

func (sp *PSQLIB) xxxx(
	tx *sql.Tx, _in_delmsgids delMsgIDState,
	modid uint64, modCC ModCombinedCaps) (
	out_delmsgids delMsgIDState, err error) {

	out_delmsgids = _in_delmsgids

	srcdir := sp.src.Main()
	xst := tx.Stmt(sp.st_prep[st_mod_fetch_and_clear_mod_msgs_continue])

	// 666 days in the future
	off_pdate := time.Now().Add(time.Hour * 24 * 666).UTC()
	off_g_p_id := uint64(0)
	off_b_id := uint32(0)

	type idt struct {
		bid  boardID
		bpid postID
	}
	type postinfo struct {
		gpid      postID
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
			err = sp.sqlError("st_web_fetch_and_clear_mod_msgs query", err)
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
				err = sp.sqlError("st_web_fetch_and_clear_mod_msgs rows scan", err)
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
			err = sp.sqlError("st_web_fetch_and_clear_mod_msgs rows it", err)
			return
		}

		for i := range posts {
			// prepare postinfo good enough for execModCmd
			pi := mailib.PostInfo{
				MessageID: CoreMsgIDStr(posts[i].msgid),
				Date:      posts[i].date_sent,
				MI: mailib.MessageInfo{
					Title:   posts[i].title,
					Message: posts[i].message,
				},
				E: mailib.PostExtraAttribs{
					TextAttachment: posts[i].txtidx,
				},
			}

			sp.log.LogPrintf(DEBUG,
				"setmodpriv: executing <%s> from board[%s]",
				posts[i].msgid, posts[i].bname)

			var delmodids delModIDState
			var inputerr bool

			out_delmsgids, delmodids, err, inputerr = sp.execModCmd(
				tx, posts[i].gpid, posts[i].xid.bid, posts[i].xid.bpid,
				modid, modCC,
				pi, posts[i].files, pi.MessageID,
				CoreMsgIDStr(posts[i].ref), out_delmsgids, delmodids)

			if err != nil {

				if inputerr {
					// mod msg is just fucked at this point
					// this shouldn't happen
					// XXX should we delete this bad msg??
					sp.log.LogPrintf(ERROR,
						"setmodpriv: [proceeding anyway] inputerr while execing <%s>: %v",
						posts[i].msgid, err)
					err = nil
					continue
				}

				// return err directly
				return
			}

			if delmodids.contain(modid) {
				sp.log.LogPrintf(
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
