package nntp

import (
	"errors"
	"fmt"
	"io"
	tp "net/textproto"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/bufreader"
	. "nekochan/lib/logx"
)

type ScraperDatabase interface {
	GetLastNewNews() (t int64, err error)
	UpdateLastNewNews(t int64) error

	GetLastNewGroups() (t int64, err error)
	UpdateLastNewGroups(t int64) error

	// MAY make new group, may return id==0 if no info about group before this
	// if id<0 then no such group currently exists
	GetGroupID(group []byte) (id int64, err error)
	UpdateGroupID(group string, id uint64) error

	// to keep list of received newsgroups
	StartTempGroups() error        // before we start adding
	CancelTempGroups()             // if we fail in middle of adding
	FinishTempGroups(partial bool) // after all list is added
	DoneTempGroups()               // after we finished using them
	StoreTempGroupID(group []byte, new_id uint64, old_id uint64) error
	StoreTempGroup(group []byte, old_id uint64) error
	LoadTempGroup() (group string, new_id int64, old_id uint64, err error)

	IsArticleWanted(msgid FullMsgIDStr) (bool, error)

	ReadArticle(r io.Reader, msgid CoreMsgIDStr) (err error, unexpected bool)
}

var errTooLargeResponse = errors.New("too large response")

type clientState struct {
	initialResponseUnderstod bool
	initialResponseAllowPost bool

	badActiveList     bool
	badNewsgroupsList bool
	badCapabilities   bool
	badHdr            bool
	badXHdr           bool
	badOver           bool
	badXOver          bool

	capHdr    bool
	capOver   bool
	capReader bool

	workaroundStupidActiveList bool
}

func (s *clientState) canHdr() bool {
	return s.capHdr && !s.badHdr
}

func (s *clientState) canXHdr() bool {
	return !s.badXHdr
}

func (s *clientState) canOver() bool {
	return s.capOver && !s.badOver
}

func (s *clientState) canXOver() bool {
	return !s.badXOver
}

type NNTPClient struct {
	inbuf [512]byte
	args  [][]byte

	w  *tp.Writer
	r  *bufreader.BufReader
	dr *bufreader.DotReader

	s   clientState
	log Logger
}

func (c *NNTPClient) openDotReader() *bufreader.DotReader {
	if c.dr == nil {
		c.dr = bufreader.NewDotReader(c.r)
	} else {
		c.dr.Reset()
	}
	return c.dr
}

func (c *NNTPClient) readLine() (incmd []byte, e error) {
	var i int
	i, e = c.r.ReadUntil(c.inbuf[:], '\n')
	if e != nil {
		if e == bufreader.ErrDelimNotFound {
			// response too large to process, error
			e = errTooLargeResponse
		}
		return
	}

	if i > 1 && c.inbuf[i-2] == '\r' {
		incmd = c.inbuf[:i-2]
	} else {
		incmd = c.inbuf[:i-1]
	}

	return
}

func parseResponseCode(line []byte) (code uint, rest []byte, err error) {
	// NNTP uses exactly 3 characters always so expect that
	if len(line) < 3 || !isNumberSlice(line[:3]) ||
		(len(line) > 3 && line[3] != ' ') {

		return 0, line, fmt.Errorf("response %q not understod", line)
	}
	code = stoi(line[:3])
	if code < 100 || code >= 600 {
		err = fmt.Errorf("response code %d out of range", code)
	}
	return code, line[3:], err
}

// parseResponseArguments parses rest of response line,
// up to specified number of arguments, appending to args slice,
// returning updated args slice and unprocessed slice of line.
// If requested num is -1 it will parse as much arguments as there are.
func parseResponseArguments(
	line []byte, num int, args [][]byte) ([][]byte, []byte) {

	if len(line) == 0 || num == 0 {
		return args, nil
	}
	i := 1 // skip initial guaranteed space
	for i < len(line) && num != 0 {
		for i < len(line) && line[i] == ' ' {
			i++
		}
		s := i
		for i < len(line) && line[i] != ' ' {
			i++
		}
		if i <= s {
			break
		}
		args = append(args, line[s:i])
		num--
	}
	return args, line[i:]
}

func (c *NNTPClient) readResponse() (
	code uint, rest []byte, err error, fatal bool) {

	incmd, err := c.readLine()
	if err != nil {
		fatal = true
		return
	}

	code, rest, err = parseResponseCode(incmd)
	return
}

func (c *NNTPClient) handleInitial() error {
	code, rest, err, _ := c.readResponse()
	if err != nil {
		return fmt.Errorf(
			"error reading initial response: %v, %q",
			err, au.TrimWSBytes(rest))
	}
	if code == 200 {
		c.s.initialResponseAllowPost = true
	} else if code == 201 {
		c.s.initialResponseAllowPost = false
	} else {
		return fmt.Errorf(
			"bad initial response %d %q",
			code, au.TrimWSBytes(rest))
	}
	return nil
}
