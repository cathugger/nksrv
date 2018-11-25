package nntp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	tp "net/textproto"
	"time"

	au "nekochan/lib/asciiutils"
	"nekochan/lib/bufreader"
	. "nekochan/lib/logx"
)

type ClientDatabase interface {
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

type scraperState struct {
	initialResponseUnderstod bool
	initialResponseAllowPost bool

	badActiveList     bool
	badNewsgroupsList bool
	badCapabilities   bool
	badOver           bool
	badXOver          bool

	capHdr    bool
	capOver   bool
	capReader bool

	workaroundStupidActiveList bool
}

func (s *scraperState) canOver() bool {
	return s.capOver && !s.badOver
}

func (s *scraperState) canXOver() bool {
	return !s.badXOver
}

type todoArticle struct {
	id    uint64
	msgid FullMsgIDStr
}

type NNTPScraper struct {
	inbuf [512]byte
	args  [][]byte

	w  *tp.Writer
	r  *bufreader.BufReader
	dr *bufreader.DotReader

	s   scraperState
	db  ClientDatabase
	log Logger

	todoList []todoArticle
}

func NewNNTPScraper(db ClientDatabase, logx LoggerX) *NNTPScraper {
	c := &NNTPScraper{db: db}
	c.log = NewLogToX(logx, fmt.Sprintf("nntpscraper.%p", c))
	return c
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
			if end == 0 {
				end = i
			}
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
	if s >= i || !FullValidGroupSlice(line[s:i]) {
		err = fmt.Errorf("bad group %q", line[s:i])
		return
	}
	name = line[s:i]

	skipWS()
	s = i
	skipNonWS()
	if s >= i || !isNumberSlice(line[s:i]) {
		err = fmt.Errorf("bad hiwm %q", line[s:i])
		return
	}
	hiwm = stoi64(line[s:i])

	skipWS()
	s = i
	skipNonWS()
	if s >= i || !isNumberSlice(line[s:i]) {
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
			c.s.badActiveList = true
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
			if c.s.workaroundStupidActiveList {
				// unless it's broke implementation
				hiwm, lowm = lowm, hiwm
			} else {
				hiwm = 0
			}
		}
		old_id, e := c.db.GetGroupID(gname)
		if e != nil {
			err = fmt.Errorf("GetGroupID() failed: %v", e)
			return
		}
		c.log.LogPrintf(DEBUG,
			"doActiveList: got existing group %q id %d", gname, old_id)
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
			c.s.badNewsgroupsList = true
			err = fmt.Errorf("failed reading list line: %v", e)
			return
		}
		old_id, e := c.db.GetGroupID(gname)
		if e != nil {
			err = fmt.Errorf("GetGroupID() failed: %v", e)
			return
		}
		c.log.LogPrintf(DEBUG,
			"doNewsgroupsList: got existing group %q id %d", gname, old_id)
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

func (c *NNTPScraper) doCapabilities() (err error, fatal bool) {
	c.log.LogPrintf(DEBUG, "querying CAPABILITIES")
	err = c.w.PrintfLine("CAPABILITIES")
	if err != nil {
		fatal = true
		return
	}
	code, _, err, fatal := c.readResponse()
	if err != nil {
		c.log.LogPrintf(DEBUG, "readResponse() err: %v", err)
		return
	}
	if code != 101 {
		c.log.LogPrintf(DEBUG, "code: %d", code)
		c.s.badCapabilities = true
		return
	}
	c.log.LogPrintf(DEBUG, "reading CAPABILITIES")
	dr := c.openDotReader()
	defer func() {
		if err != nil {
			dr.Discard(-1)
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
		c.log.LogPrintf(DEBUG, "got capability line %q", line)
		x := parseKeyword(line)
		capability := unsafeBytesToStr(line[:x])
		switch capability {
		case "HDR":
			c.s.capHdr = true
		case "OVER":
			c.s.capOver = true
		case "READER":
			c.s.capReader = true
		case "IMPLEMENTATION":
			c.args, _ = parseResponseArguments(line[x:], 6, c.args[:0])
			if len(c.args) != 0 {
				impl := unsafeBytesToStr(c.args[0])
				if au.EqualFoldString(impl, "SRNDv2") {
					c.log.LogPrintf(INFO, "detected SRNDv2")
					// workarounds for some jeff' stuff
					c.s.workaroundStupidActiveList = true
				}
			}
		}
	}
	// done
	c.log.LogPrintf(DEBUG, "done readin CAPABILITIES")
	return
}

func (c *NNTPScraper) Run(network, address string) {
	// TODO
	for {
		c.log.LogPrintf(DEBUG, "dialing...")
		conn, e := net.Dial(network, address)
		if e != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		c.s = scraperState{}
		c.w = tp.NewWriter(bufio.NewWriter(conn))
		c.r = bufreader.NewBufReader(conn)
		c.dr = nil
		c.log.LogPrintf(DEBUG, "scraping...")
		e = c.main()
		conn.Close()
		if e != nil {
			c.log.LogPrintf(WARN, "scraper error: %v", e)
		} else {
			c.log.LogPrintf(WARN, "scraper done")
		}
		time.Sleep(10 * time.Second)
	}
}

func (c *NNTPScraper) doGroup(
	gname string) (new_id int64, err error, notexists, fatal bool) {

	err = c.w.PrintfLine("GROUP %s", gname)
	if err != nil {
		fatal = true
		return
	}

	code, rest, err, fatal := c.readResponse()
	if err != nil {
		c.log.LogPrintf(DEBUG, "readResponse() err: %v", err)
		return
	}

	if code == 211 {

		c.args, _ = parseResponseArguments(rest, 4, c.args[:0])
		if len(c.args) < 3 ||
			!isNumberSlice(c.args[0]) ||
			!isNumberSlice(c.args[1]) ||
			!isNumberSlice(c.args[2]) {

			return -1, fmt.Errorf(
				"bad successful group response %q",
				au.TrimWSBytes(rest)), false, false
		}

		num := stoi64(c.args[0])
		lo := stoi64(c.args[1])
		hi := stoi64(c.args[2])

		c.args = c.args[:0]

		if lo > hi || num == 0 {
			// empty group
			hi = 0
		}
		new_id = int64(hi) // we need only high id
		return

	} else if code == 411 {
		return -1, errors.New("no such newsgroup"), true, false
	} else {
		return -1, fmt.Errorf(
			"bad GROUP err %d %q",
			code, au.TrimWSBytes(rest)), false, false
	}
}

func (c *NNTPScraper) getOverLineInfo(
	dr *bufreader.DotReader) (
	id uint64, msgid FullMsgID, err error) {

	i := 0
	nomore := false
	eatField := func() (field []byte, err error) {
		s := i
		for {
			b, e := dr.ReadByte()
			if e != nil {
				err = e
				return
			}
			if b == '\n' {
				field = c.inbuf[s:i]
				nomore = true
				return
			}
			if b == '\t' {
				field = c.inbuf[s:i]
				return
			}
			if i >= len(c.inbuf) {
				err = errTooLargeResponse
				return
			}
			c.inbuf[i] = b
			i++
		}
	}
	ignoreField := func() (err error) {
		for {
			b, e := dr.ReadByte()
			if e != nil {
				err = e
				return
			}
			if b == '\n' {
				nomore = true
				return
			}
			if b == '\t' {
				return
			}
		}
	}

	defer func() {
		if !nomore {
			for {
				b, e := dr.ReadByte()
				if e != nil || b == '\n' {
					return
				}
			}
		}
	}()

	// {RFC 2980}
	// (article number goes before these, ofc)
	// The sequence of fields must be in this order:
	// subject, author, date, message-id, references,
	// byte count, and line count.

	// number
	snum, err := eatField()
	if err != nil || nomore {
		return
	}
	snum = au.TrimWSBytes(snum)
	if len(snum) == 0 || !isNumberSlice(snum) {
		err = fmt.Errorf("bad id %q", snum)
		return
	}
	id = stoi64(snum)
	// subject, author, date
	for xx := 0; xx < 3; xx++ {
		err = ignoreField()
		if err != nil {
			return
		}
		if nomore {
			err = errors.New("wanted more fields")
			return
		}
	}
	// message-id
	smsgid, err := eatField()
	if err != nil {
		return
	}
	smsgid = au.TrimWSBytes(smsgid)
	msgid = FullMsgID(smsgid)
	if !ValidMessageID(msgid) {
		err = fmt.Errorf("invalid msg-id %q", smsgid)
		return
	}

	return
}

func (c *NNTPScraper) eatArticle(msgid FullMsgIDStr) (err error, fatal bool) {
	dr := c.openDotReader()
	defer func() {
		if err != nil {
			dr.Discard(-1)
		}
	}()

	err, fatal = c.db.ReadArticle(dr, CutMsgIDStr(msgid))
	if err != nil {
		if fatal {
			c.log.LogPrintf(ERROR, "c.db.ReadArticle fatal err: %v", err)
		} else {
			c.log.LogPrintf(ERROR, "c.db.ReadArticle expected err: %v", err)
		}
	}
	return
}

func (c *NNTPScraper) processTODOList(
	group string, maxid int64) (new_maxid int64, err error, fatal bool) {

	new_maxid = -1
	defer func() {
		c.log.LogPrintf(DEBUG, "processTODOList defer: maxid(%d) new_maxid(%d)",
			maxid, new_maxid)
		if new_maxid >= 0 && new_maxid > maxid {
			c.log.LogPrintf(DEBUG, "processTODOList defer: updating group id")
			c.db.UpdateGroupID(group, uint64(new_maxid))
		}
	}()

	c.log.LogPrintf(DEBUG, "start TODO list")
	for i := range c.todoList {
		wanted, e := c.db.IsArticleWanted(c.todoList[i].msgid)
		if e != nil {
			c.log.LogPrintf(ERROR,
				"IsArticleWanted(%s) fail: %v", c.todoList[i].msgid, e)
			err = e
			return
		}

		if !wanted {
			c.log.LogPrintf(DEBUG, "TODO list %d %s unwanted",
				c.todoList[i].id, c.todoList[i].msgid)

			if int64(c.todoList[i].id) > new_maxid {
				new_maxid = int64(c.todoList[i].id)
			}

			continue
		}
		c.log.LogPrintf(DEBUG, "TODO list %d %s wanted",
			c.todoList[i].id, c.todoList[i].msgid)

		// we want it, so ask for it
		err = c.w.PrintfLine("ARTICLE %d", c.todoList[i].id)
		if err != nil {
			fatal = true
			return
		}

		var code uint
		var rest []byte
		code, rest, err, fatal = c.readResponse()
		if err != nil {
			c.log.LogPrintf(DEBUG, "readResponse() err: %v", err)
			return
		}

		if code == 220 {
			// we have to process it now
			// -->>
		} else if code == 423 || code == 430 {
			// article gone..
			c.log.LogPrintf(WARN,
				"processTODOList: negative ARTICLE response %d %q",
				code, au.TrimWSBytes(rest))
			continue
		} else {
			c.log.LogPrintf(WARN,
				"processTODOList: weird ARTICLE response %d %q",
				code, au.TrimWSBytes(rest))
			continue
		}
		// process article
		err, fatal = c.eatArticle(c.todoList[i].msgid)
		if err != nil {
			if fatal {
				return
			} else {
				c.log.LogPrintf(WARN,
					"processTODOList: eatArticle(%s) fail: %v",
					c.todoList[i].msgid, err)
				err = nil
			}
		}
		// we ate it successfuly
		if int64(c.todoList[i].id) > new_maxid {
			new_maxid = int64(c.todoList[i].id)
		}
	}
	c.log.LogPrintf(DEBUG, "end TODO list")
	return
}

func (c *NNTPScraper) eatGroupSlice(
	group string, r_begin, r_end, maxid int64) (new_maxid int64, err error, fatal bool) {

	overD := false

	printOverLine := func(over string) error {
		if r_end >= 0 {
			if r_begin != r_end {
				return c.w.PrintfLine("%s %d-%d", over, r_begin, r_end)
			} else {
				return c.w.PrintfLine("%s %d", over, r_begin)
			}
		} else {
			return c.w.PrintfLine("%s %d-", over, r_begin)
		}
	}

	if c.s.canOver() {
		err = printOverLine("OVER")
		if err != nil {
			fatal = true
			return
		}

		var code uint
		var rest []byte
		code, rest, err, fatal = c.readResponse()
		if err != nil {
			c.log.LogPrintf(DEBUG, "readResponse() err: %v", err)
			return
		}
		if code == 224 {
			// ayy it's all gucci
			// common code path will take care of this
			overD = true
		} else if code == 423 || code == 420 {
			// can happen
			return
		} else {
			c.log.LogPrintf(WARN,
				"unexpected OVER response %d %q, falling back to XOVER",
				code, au.TrimWSBytes(rest))
			c.s.badOver = true
		}
	}
	if !overD && c.s.canXOver() {
		err = printOverLine("XOVER")
		if err != nil {
			fatal = true
			return
		}

		var code uint
		var rest []byte
		code, rest, err, fatal = c.readResponse()
		if err != nil {
			c.log.LogPrintf(DEBUG, "readResponse() err: %v", err)
			return
		}
		if code == 224 {
			// ayy it's all gucci
			overD = true
		} else if code == 423 || code == 420 {
			// can happen
			return
		} else {
			c.log.LogPrintf(WARN,
				"unexpected XOVER response %d %q",
				code, au.TrimWSBytes(rest))
			c.s.badXOver = true
		}
	}
	if !overD {
		err = errors.New("can't use OVER or XOVER")
		return
	}

	// uhh... gotta parse OVER/XOVER lines now..
	dr := c.openDotReader()
	defer func() {
		if err != nil {
			dr.Discard(-1)
		}
	}()
	c.todoList = c.todoList[:0] // reuse
	for {
		var id uint64
		var msgid FullMsgID
		id, msgid, err = c.getOverLineInfo(dr)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
		// if we didn't ask for this, don't include
		if id == 0 || id < uint64(r_begin) ||
			(r_end >= 0 && id > uint64(r_end)) ||
			(r_end < 0 && id > uint64(r_begin)+899) {

			continue
		}

		if len(c.todoList) >= 900 {
			// safeguard
			continue
		}

		// add to list to query
		c.todoList = append(c.todoList, todoArticle{
			id: id, msgid: FullMsgIDStr(msgid)})
	}
	// loaded list.. now process it
	new_maxid, err, fatal = c.processTODOList(group, maxid)
	return
}

func (c *NNTPScraper) eatGroup(
	group string, old_id, new_id uint64) (err error, fatal bool) {

	var r_begin, r_end int64

	r_begin = int64(old_id) + 1

	if new_id > uint64(r_begin)+599 {
		r_end = r_begin + 599
	} else {
		r_end = -1
	}
	maxid := int64(old_id)
	for {
		maxid, err, fatal = c.eatGroupSlice(group, r_begin, r_end, maxid)
		if err != nil {
			return
		}
		if r_end < 0 {
			// this was supposed to be last one
			break
		}
		r_begin = r_end + 1
		if uint64(r_begin) > new_id {
			break
		}
		if new_id > uint64(r_begin)+599 {
			r_end = r_begin + 599
		} else {
			r_end = -1
		}
	}
	return
}

func (c *NNTPScraper) main() error {
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

	e, fatal := c.doCapabilities()
	if e != nil {
		if fatal {
			return fmt.Errorf("doCapabilities() failed: %v", e)
		} else {
			c.log.LogPrintf(WARN, "doCapabilities() failed: %v", e)
		}
	}

	if !c.s.capReader {
		err = c.w.PrintfLine("MODE READER")
		if err != nil {
			return fmt.Errorf("error writing mode-reader command: %v", err)
		}
		code, rest, err, fatal := c.readResponse()
		if err == nil {
			if code == 200 {
				c.s.initialResponseAllowPost = true
			} else if code > 200 && code < 300 {
				c.s.initialResponseAllowPost = false
			} else if code == 502 {
				return fmt.Errorf(
					"bad mode-reader response %d %q", code, au.TrimWSBytes(rest))
			} else if code == 500 || code == 501 {
				// do nothing if not implemented
			} else {
				c.log.LogPrintf(WARN, "weird mode-reader response %d %q",
					code, au.TrimWSBytes(rest))
			}
		} else {
			if fatal {
				return fmt.Errorf("error reading mode-reader response: %v", err)
			} else {
				c.log.LogPrintf(WARN, "error reading mode-reader response: %v", e)
			}
		}
	}

	gotGroupList := false
	if !gotGroupList && !c.s.badActiveList {
		e, fatal = c.doActiveList()
		if e != nil {
			if fatal {
				return fmt.Errorf("doActiveList method failed: %v", e)
			} else {
				c.log.LogPrintf(WARN, "doActiveList method failed: %v", e)
			}
		} else {
			gotGroupList = true
		}
	}
	if !gotGroupList && !c.s.badNewsgroupsList {
		e, fatal = c.doNewsgroupsList()
		if e != nil {
			if fatal {
				return fmt.Errorf("doNewsgroupsList method failed: %v", e)
			} else {
				c.log.LogPrintf(WARN, "doNewsgroupsList method failed: %v", e)
			}
		} else {
			gotGroupList = true
		}
	}
	if !gotGroupList {
		return errors.New("no methods left to get group list")
	}

	c.log.LogPrintf(DEBUG, "scraper will load temp groups")
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

		if new_id >= 0 && old_id >= uint64(new_id) {
			if old_id != uint64(new_id) {
				// keep track of reduction too
				c.db.UpdateGroupID(group, uint64(new_id))
			}
			// skip this
			continue
		}

		var g_id int64
		var notexists bool
		g_id, err, notexists, fatal = c.doGroup(group)
		if err != nil && !notexists {
			if fatal {
				return fmt.Errorf("doGroup failed: %v", e)
			} else {
				c.log.LogPrintf(WARN, "doGroup failed: %v", e)
			}
			// next group, I guess..
			continue
		}
		if new_id < 0 || g_id > new_id {
			new_id = g_id
		}

		err, fatal = c.eatGroup(group, old_id, uint64(new_id))
		if err != nil {
			if fatal {
				return fmt.Errorf("eatGroup failed: %v", err)
			} else {
				c.log.LogPrintf(WARN, "eatGroup failed: %v", err)
			}
		}
	}

	c.db.DoneTempGroups()

	// amount of arguments is defined by response code
	return nil // TODO
}
