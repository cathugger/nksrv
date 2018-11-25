package nntp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	tp "net/textproto"
	"time"

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

	ReadArticle(r io.Reader, msgid CoreMsgIDStr, expectedgroup string) (
		err error, unexpected bool, wantedroot FullMsgIDStr)
}

type todoArticle struct {
	id    uint64
	msgid FullMsgIDStr
}

type NNTPScraper struct {
	NNTPClient

	db       ScraperDatabase
	todoList []todoArticle
}

func NewNNTPScraper(db ScraperDatabase, logx LoggerX) *NNTPScraper {
	c := &NNTPScraper{db: db}
	c.log = NewLogToX(logx, fmt.Sprintf("nntpscraper.%p", c))
	return c
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
		// safeguard
		if int64(hiwm) < 0 {
			hiwm = math.MaxInt64
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
		c.s = clientState{}
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

		var num, lo, hi uint64
		num, lo, hi, err = c.parseGroupResponse(rest)
		if err != nil {
			return
		}

		if lo > hi || num == 0 {
			// empty group
			hi = 0
		}
		// safeguard
		if int64(hi) < 0 {
			hi = math.MaxInt64
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

func (c *NNTPScraper) eatArticle(
	msgid FullMsgIDStr, expectedgroup string) (
	err error, fatal bool, wantroot FullMsgIDStr) {

	dr := c.openDotReader()
	defer func() {
		if err != nil {
			dr.Discard(-1)
		}
	}()

	//c.log.LogPrintf(DEBUG, "eatArticle: inside")

	err, fatal, wantroot = c.db.ReadArticle(
		dr, CutMsgIDStr(msgid), expectedgroup)

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
		err, fatal, _ = c.eatArticle(c.todoList[i].msgid, group)
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

func (c *NNTPScraper) eatHdrOutput(
	r_begin, r_end int64) (err error) {

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

		id, msgid, err = c.eatHdrMsgIDLine(dr)
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

	return
}

func (c *NNTPScraper) eatOverOutput(
	r_begin, r_end int64) (err error) {

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

	return
}

func (c *NNTPScraper) eatGroupSlice(
	group string, r_begin, r_end, maxid int64) (
	new_maxid int64, err error, fatal bool) {

	printHdrLine := func(hdr string) error {
		if r_end >= 0 {
			if r_begin != r_end {
				return c.w.PrintfLine("%s Message-ID %d-%d", hdr, r_begin, r_end)
			} else {
				return c.w.PrintfLine("%s Message-ID %d", hdr, r_begin)
			}
		} else {
			return c.w.PrintfLine("%s Message-ID %d-", hdr, r_begin)
		}
	}

	hdrD := false

	if c.s.canHdr() {
		err = printHdrLine("HDR")
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
		if code == 225 || code == 221 {
			// ayy it's all gucci
			// common code path will take care of this
			hdrD = true
		} else if code == 423 || code == 420 {
			// can happen
			return
		} else {
			c.log.LogPrintf(WARN,
				"unexpected HDR response %d %q, falling back to XHDR",
				code, au.TrimWSBytes(rest))
			c.s.badHdr = true
		}
	}
	if !hdrD && c.s.canXHdr() {
		err = printHdrLine("XHDR")
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
		if code == 225 || code == 221 {
			// ayy it's all gucci
			// common code path will take care of this
			hdrD = true
		} else if code == 423 || code == 420 {
			// can happen
			return
		} else {
			c.log.LogPrintf(WARN,
				"unexpected XHDR response %d %q",
				code, au.TrimWSBytes(rest))
			c.s.badXHdr = true
		}
	}
	if hdrD {
		// parse HDR/XHDR lines
		err = c.eatHdrOutput(r_begin, r_end)
		// loaded list.. now process it
		new_maxid, err, fatal = c.processTODOList(group, maxid)
		return
	}

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

	overD := false

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
	if overD {
		// uhh... gotta parse OVER/XOVER lines now..
		err = c.eatOverOutput(r_begin, r_end)
		// loaded list.. now process it
		new_maxid, err, fatal = c.processTODOList(group, maxid)
		return
	}

	err = errors.New(
		"can't list group slice (tried HDR/XHDR/OVER/XOVER)")
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
	var e error

	e = c.handleInitial()
	if e != nil {
		return e
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
		e = c.w.PrintfLine("MODE READER")
		if e != nil {
			return fmt.Errorf("error writing mode-reader command: %v", e)
		}
		code, rest, e, fatal := c.readResponse()
		if e == nil {
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
				return fmt.Errorf("error reading mode-reader response: %v", e)
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

		// maybe we don't need to bother with this group
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
		g_id, e, notexists, fatal = c.doGroup(group)
		if e != nil && !notexists {
			if fatal {
				return fmt.Errorf("doGroup failed: %v", e)
			} else {
				c.log.LogPrintf(WARN, "doGroup failed: %v", e)
			}
			// next group, I guess..
			continue
		}
		// in case we had no info about new_id before, or it's higher..
		if new_id < 0 || g_id > new_id {
			new_id = g_id

			// recheck
			if old_id >= uint64(new_id) {
				if old_id != uint64(new_id) {
					// keep track of reduction too
					c.db.UpdateGroupID(group, uint64(new_id))
				}
				// skip this
				continue
			}
		}

		e, fatal = c.eatGroup(group, old_id, uint64(new_id))
		if e != nil {
			if fatal {
				return fmt.Errorf("eatGroup failed: %v", e)
			} else {
				c.log.LogPrintf(WARN, "eatGroup failed: %v", e)
			}
		}
	}

	c.db.DoneTempGroups()

	// amount of arguments is defined by response code
	return nil // TODO
}
