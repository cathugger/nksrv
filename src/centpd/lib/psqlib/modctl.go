package psqlib

import (
	"database/sql"
	"io"
	"os"
	"strings"

	"centpd/lib/bufreader"
	"centpd/lib/mailib"
	mm "centpd/lib/minimail"
)

func (sp *PSQLIB) modCmdDelete(
	tx *sql.Tx, gpid postID, bid boardID, bpid postID,
	pi mailib.PostInfo, selfid, ref CoreMsgIDStr,
	cmd string, args []string) (err error) {

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

	err = sp.banByMsgID(tx, cmsgids, bid, bpid, pi.MI.Title)
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
	modid int64, modpriv ModPriv,
	pi mailib.PostInfo, filenames []string,
	selfid, ref CoreMsgIDStr) (err error) {

	r, c, err := getModCmdInput(pi, filenames)
	if err != nil {
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

		unsafe_line := unsafeBytesToStr(linebuf[:read])
		unsafe_fields := strings.Fields(unsafe_line)

		if len(unsafe_fields) != 0 {

			cmd := strings.ToLower(unsafe_fields[0])
			args := unsafe_fields[1:]

			// TODO log commands we couldn't understand
			switch cmd {
			case "delete":
				if modpriv >= ModPrivMod {
					// global delete by msgid
					err = sp.modCmdDelete(tx, gpid, bid, bpid, pi, selfid, ref, cmd, args)
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

	return nil
}
