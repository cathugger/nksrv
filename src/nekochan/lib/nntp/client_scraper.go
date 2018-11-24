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

type ClientDatabase interface {
	GetNodeTime() (t int64, err error)
	UpdateNodeTime(t int64) error

	// MAY make new group, may return id==0 if no info about group before this
	// if id<0 then no such group currently exists
	GetGroupID(group []byte) (id int64, err error)
	UpdateGroupID(group []byte, id uint64) error

	// to keep list of received newsgroups
	StartTempGroups() error        // before we start adding
	CancelTempGroups()             // if we fail in middle of adding
	FinishTempGroups(partial bool) // after all list is added
	DoneTempGroups()               // after we finished using them
	StoreTempGroupID(group []byte, new_id uint64, old_id uint64) error
	StoreTempGroup(group []byte, old_id uint64) error
	LoadTempGroup() (group string, new_id int64, old_id uint64, err error)

	ReadArticle(r io.Reader, msgid CoreMsgID) (err error, unexpected bool)
}

type scraperState struct {
	initialResponseUnderstod bool
	initialResponseAllowPost bool
	badActiveList            bool
	badNewsgroupsList        bool
}

type NNTPScraper struct {
	inbuf [512]byte

	w  *tp.Writer
	r  *bufreader.BufReader
	dr *bufreader.DotReader

	s   scraperState
	db  ClientDatabase
	log Logger
}

func (c *NNTPScraper) openDotReader() *bufreader.DotReader {
	if c.dr == nil {
		c.dr = bufreader.NewDotReader(c.r)
	} else {
		c.dr.Reset()
	}
	return c.dr
}

var errTooLargeResponse = errors.New("too large response")

func (c *NNTPScraper) readLine() (incmd []byte, e error) {
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

func (c *NNTPScraper) readResponse() (
	code uint, rest []byte, err error, fatal bool) {

	incmd, err := c.readLine()
	if err != nil {
		fatal = true
		return
	}

	code, rest, err = parseResponseCode(incmd)
	return
}

func (c *NNTPScraper) readDotLine(dr *bufreader.DotReader) ([]byte, error) {
	i := 0
	for {
		b, e := dr.ReadByte()
		if e != nil {
			return c.inbuf[:i], e
		}
		if b == '\n' {
			return c.inbuf[:i], nil
		}
		if i >= len(c.inbuf) {
			return c.inbuf[:i], errTooLargeResponse
		}
		c.inbuf[i] = b
		i++
	}
}

func (c *NNTPScraper) readOnlyNewsgroup(
	dr *bufreader.DotReader) ([]byte, error) {

	i := 0
	end := 0
	for {
		b, e := dr.ReadByte()
		if e != nil {
			return c.inbuf[:end], e
		}
		if b == '\n' {
			if end == 0 || !FullValidGroupSlice(c.inbuf[:end]) {
				return nil, fmt.Errorf("bad group %q", c.inbuf[:end])
			}
			return c.inbuf[:end], nil
		}
		if end == 0 {
			if b == ' ' || b == '\t' {
				end = i
				continue
			}
			if i >= len(c.inbuf) {
				return nil, errTooLargeResponse
			}
			c.inbuf[i] = b
			i++
		}
	}
}

func parseListActiveLine(
	line []byte) (name []byte, hiwm, lowm uint64, status []byte, err error) {

	i := 0
	skipWS := func() {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
	}
	skipNonWS := func() {
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			i++
		}
	}

	//skipWS()
	s := i
	skipNonWS()
	if s <= i || !FullValidGroupSlice(line[s:i]) {
		err = fmt.Errorf("bad group %q", line[s:i])
		return
	}
	name = line[s:i]

	skipWS()
	s = i
	skipNonWS()
	if s <= i || !isNumberSlice(line[s:i]) {
		err = fmt.Errorf("bad hiwm %q", line[s:i])
		return
	}
	hiwm = stoi64(line[s:i])

	skipWS()
	s = i
	skipNonWS()
	if s <= i || !isNumberSlice(line[s:i]) {
		err = fmt.Errorf("bad lowm %q", line[s:i])
		return
	}
	lowm = stoi64(line[s:i])

	skipWS()
	s = i
	skipNonWS()
	// can be empty I guess... I don't see why not
	status = line[s:i]

	// treat any extra as error
	skipWS()
	if i < len(line) {
		err = fmt.Errorf("unknown extra data: %q", line[i:])
		return
	}

	return
}

func (c *NNTPScraper) doActiveList() (err error, fatal bool) {
	err = c.w.PrintfLine("LIST")
	if err != nil {
		fatal = true
		return
	}
	code, rest, err, fatal := c.readResponse()
	if err != nil {
		return
	}
	if code != 215 {
		c.s.badActiveList = true
		err = fmt.Errorf(
			"bad response from list %d %q",
			code, au.TrimWSBytes(rest))
		return
	}

	dr := c.openDotReader()
	defer func() {
		if err != nil {
			dr.Discard(-1)
		}
	}()

	e := c.db.StartTempGroups()
	if e != nil {
		err = fmt.Errorf("StartTempGroups() failed: %v", e)
		return
	}
	defer func() {
		if err == nil {
			c.db.FinishTempGroups(false)
		} else {
			c.db.CancelTempGroups()
		}
	}()

	for {
		line, e := c.readDotLine(dr)
		if e != nil {
			if e == io.EOF {
				break
			}
			err = fmt.Errorf("failed reading list line: %v", e)
			return
		}
		gname, hiwm, lowm, _, e := parseListActiveLine(line)
		if e != nil {
			c.s.badActiveList = true
			err = fmt.Errorf("failed parsing list line: %v", e)
			return
		}
		if hiwm < lowm {
			// negative count = no articles
			hiwm = 0
		}
		old_id, e := c.db.GetGroupID(gname)
		if e != nil {
			err = fmt.Errorf("GetGroupID() failed: %v", e)
			return
		}
		if old_id < 0 {
			// such group currently does not exist and wasn't created
			continue
		}
		e = c.db.StoreTempGroupID(gname, hiwm, uint64(old_id))
		if e != nil {
			err = fmt.Errorf("StoreTempGroup() failed: %v", e)
			return
		}
	}

	// done
	return
}

func (c *NNTPScraper) doNewsgroupsList() (err error, fatal bool) {
	err = c.w.PrintfLine("LIST NEWSGROUPS")
	if err != nil {
		fatal = true
		return
	}
	code, rest, err, fatal := c.readResponse()
	if err != nil {
		return
	}
	if code != 215 {
		c.s.badNewsgroupsList = true
		err = fmt.Errorf(
			"bad response from list %d %q",
			code, au.TrimWSBytes(rest))
		return
	}

	dr := c.openDotReader()
	defer func() {
		if err != nil {
			dr.Discard(-1)
		}
	}()

	e := c.db.StartTempGroups()
	if e != nil {
		err = fmt.Errorf("StartTempGroups() failed: %v", e)
		return
	}
	defer func() {
		if err == nil {
			c.db.FinishTempGroups(false)
		} else {
			c.db.CancelTempGroups()
		}
	}()

	for {
		gname, e := c.readOnlyNewsgroup(dr)
		if e != nil {
			if e == io.EOF {
				break
			}
			err = fmt.Errorf("failed reading list line: %v", e)
			return
		}
		old_id, e := c.db.GetGroupID(gname)
		if e != nil {
			err = fmt.Errorf("GetGroupID() failed: %v", e)
			return
		}
		if old_id < 0 {
			continue
		}
		e = c.db.StoreTempGroup(gname, uint64(old_id))
		if e != nil {
			err = fmt.Errorf("StoreTempGroup() failed: %v", e)
			return
		}
	}

	// done
	return
}

func (c *NNTPScraper) Run() error {
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

	e, fatal := c.doActiveList()
	if e != nil {
		if fatal {
			return fmt.Errorf("doActiveList method failed: %v", e)
		} else {
			c.log.LogPrintf(WARN, "doActiveList method failed: %v", e)
		}
		e, fatal = c.doNewsgroupsList()
		if e != nil {
			if fatal {
				return fmt.Errorf("doNewsgroupsList method failed: %v", e)
			} else {
				c.log.LogPrintf(WARN, "doNewsgroupsList method failed: %v", e)
			}
			return errors.New("no methods left to get group list")
		}
	}

	for {
		group, new_id, old_id, e := c.db.LoadTempGroup()
		if e != nil {
			if e == io.EOF {
				break
			}
			c.log.LogPrintf(WARN, "LoadTempGroup() failed: %v", e)
			break
		}
		c.log.LogPrintf(DEBUG, "LoadTempGroup(): g:%q n:%d o:%d",
			group, new_id, old_id)
	}

	c.db.DoneTempGroups()

	// amount of arguments is defined by response code
	return nil // TODO
}
