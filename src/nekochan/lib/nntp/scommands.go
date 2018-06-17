package nntp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type commandFunc func(c *ConnState, args [][]byte, rest []byte) bool

type command struct {
	cmdfunc    commandFunc
	minargs    int
	maxargs    int
	allowextra bool
	help       string
}

var commandMap map[string]*command
var commandList []string

var listCommandMap map[string]*command
var listCommandList []string

func init() {
	var i int

	commandMap = map[string]*command{
		"": &command{
			cmdfunc:    cmdVoid,
			allowextra: true,
		},
		"CAPABILITIES": &command{
			cmdfunc:    cmdCapabilities,
			allowextra: true,
			help:       "- print server's capabilities.",
		},
		"HELP": &command{
			cmdfunc: cmdHelp,
			help:    "- print manual.",
		},
		"LIST": &command{
			cmdfunc:    cmdList,
			allowextra: true,
			help:       "[keyword [wildmat|argument]] - query information. keyword defaults to ACTIVE.",
		},
		"QUIT": &command{
			cmdfunc:    cmdQuit,
			allowextra: true,
			help:       "- terminate connection.",
		},
		"SLAVE": &command{
			cmdfunc: cmdSlave,
			help:    "- notify server about slave status.",
		},
		"DATE": &command{
			cmdfunc: cmdDate,
			help:    "- get server's current Coordinated Universal Time.",
		},

		"GROUP": &command{
			cmdfunc: cmdGroup,
			minargs: 1,
			maxargs: 1,
			help:    "group - select current group and set current article number to first article in the group.",
		},
		"LISTGROUP": &command{
			cmdfunc: cmdListGroup,
			maxargs: 2,
			help:    "[group [range]] - select current group (if specified) and set current article number to first article in the group (even if group is not specified). list articles present in the group, optionally limited by range argument.",
		},
		"NEXT": &command{
			cmdfunc: cmdNext,
			help:    "- advance current article number to next article (if available).",
		},
		"LAST": &command{
			cmdfunc: cmdLast,
			help:    "- change current article number to previous article (if available).",
		},

		"ARTICLE": &command{
			cmdfunc: func(c *ConnState, args [][]byte, rest []byte) bool {
				commonArticleHandler(c, articleFull, args)
				return true
			},
			maxargs: 1,
			help:    "[<message-id>|number] - display the header, a blank line, then the body of the specified (or current) article.",
		},
		"HEAD": &command{
			cmdfunc: func(c *ConnState, args [][]byte, rest []byte) bool {
				commonArticleHandler(c, articleHead, args)
				return true
			},
			maxargs: 1,
			help:    "[<message-id>|number] - display the header of the specified (or current) article.",
		},
		"BODY": &command{
			cmdfunc: func(c *ConnState, args [][]byte, rest []byte) bool {
				commonArticleHandler(c, articleBody, args)
				return true
			},
			maxargs: 1,
			help:    "[<message-id>|number] - display the body of the specified (or current) article.",
		},
		"STAT": &command{
			cmdfunc: func(c *ConnState, args [][]byte, rest []byte) bool {
				commonArticleHandler(c, articleStat, args)
				return true
			},
			maxargs: 1,
			help:    "[<message-id>|number] - check existence of the specified (or current) article.",
		},

		"NEWGROUPS": &command{
			cmdfunc: cmdNewGroups,
			minargs: 2,
			maxargs: 4, // <distributions> {RFC 977}
			help:    "[YY]YYMMDD hhmmss [GMT] - list newsgroups created since specified date.",
		},
	}

	listCommandMap = map[string]*command{
		"ACTIVE": &command{
			cmdfunc: listCmdActive,
			maxargs: 1,
			help:    "[wildmat] - list valid newsgroups and associated information. returns list in format `<name> <high watermark> <low watermark> <status>`.",
		},
		"NEWSGROUPS": &command{
			cmdfunc: listCmdNewsgroups,
			maxargs: 1,
			help:    "[wildmat] - list newsgroups and their descriptions. returns list in format `<name> <description>`. usually separated by tab. description may contain spaces.",
		},
		"OVERVIEW.FMT": &command{
			cmdfunc: listCmdOverviewFmt,
			help:    "- list metadata fields returned by OVER command",
		},
	}

	commandList = make([]string, len(commandMap))
	i = 0
	for k := range commandMap {
		commandList[i] = k
		i++
	}
	sort.Strings(commandList)

	listCommandList = make([]string, len(listCommandMap))
	i = 0
	for k := range listCommandMap {
		listCommandList[i] = k
		i++
	}
	sort.Strings(listCommandList)
}

func cmdHelp(c *ConnState, args [][]byte, rest []byte) bool {
	c.w.PrintfLine("100 here's manual")
	dw := c.w.DotWriter()
	for _, k := range commandList {
		cmd := commandMap[k]
		if cmd.help != "" {
			fmt.Fprintf(dw, "%s %s\n", k, cmd.help)
		}
		if k == "LIST" {
			for _, lk := range listCommandList {
				lcmd := listCommandMap[lk]
				if lcmd.help != "" {
					fmt.Fprintf(dw, "LIST %s %s\n", lk, lcmd.help)
				}
			}
		}
	}
	dw.Close()
	return true
}

func cmdQuit(c *ConnState, args [][]byte, rest []byte) bool {
	c.w.PrintfLine("205 goodbye.")
	// will close gracefuly
	return false
}

func cmdDate(c *ConnState, args [][]byte, rest []byte) bool {
	t := time.Now().UTC()
	Y, M, D := t.Date()
	h, m, s := t.Clock()
	// 111 YYYYMMDDhhmmss    Server date and time
	// XXX will break when year>9999
	c.w.PrintfLine("111 %4d%2d%2d%2d%2d%2d YYYYMMDDhhmmss", Y, M, D, h, m, s)
	return true
}

func cmdCapabilities(c *ConnState, args [][]byte, rest []byte) bool {
	c.w.PrintfLine("101 capability list follows")
	dw := c.w.DotWriter()
	fmt.Fprintf(dw, "VERSION 2\n")
	fmt.Fprintf(dw, "READER\n")
	fmt.Fprintf(dw, "IHAVE\n")
	//fmt.Fprintf(dw, "NEWNEWS\n")
	fmt.Fprintf(dw, "OVER\n")
	// TODO
	dw.Close()
	return true
}

func cmdSlave(c *ConnState, args [][]byte, rest []byte) bool {
	c.w.PrintfLine("202 slave status noted") // :^)
	return true
}

const (
	articleFull = iota
	articleHead
	articleBody
	articleStat
	articleAmmount
)

var setA = [articleAmmount]struct {
	byMsgID func(c *ConnState, msgid []byte) bool
	byNum   func(c *ConnState, num uint64) bool
	byCurr  func(c *ConnState) bool
}{
	{
		func(c *ConnState, msgid []byte) bool { return c.prov.GetArticleFullByMsgID(c.w, msgid) },
		func(c *ConnState, num uint64) bool { return c.prov.GetArticleFullByNum(c.w, c, num) },
		func(c *ConnState) bool { return c.prov.GetArticleFullByCurr(c.w, c) },
	}, {
		func(c *ConnState, msgid []byte) bool { return c.prov.GetArticleHeadByMsgID(c.w, msgid) },
		func(c *ConnState, num uint64) bool { return c.prov.GetArticleHeadByNum(c.w, c, num) },
		func(c *ConnState) bool { return c.prov.GetArticleHeadByCurr(c.w, c) },
	}, {
		func(c *ConnState, msgid []byte) bool { return c.prov.GetArticleBodyByMsgID(c.w, msgid) },
		func(c *ConnState, num uint64) bool { return c.prov.GetArticleBodyByNum(c.w, c, num) },
		func(c *ConnState) bool { return c.prov.GetArticleBodyByCurr(c.w, c) },
	}, {
		func(c *ConnState, msgid []byte) bool { return c.prov.GetArticleStatByMsgID(c.w, msgid) },
		func(c *ConnState, num uint64) bool { return c.prov.GetArticleStatByNum(c.w, c, num) },
		func(c *ConnState) bool { return c.prov.GetArticleStatByCurr(c.w, c) },
	},
}

func isPrintableASCIISlice(s []byte, e byte) bool {
	for _, c := range s {
		if c < 32 || c >= 127 || c == e {
			return false
		}
	}
	return true
}

func validMessageID(id []byte) bool {
	return len(id) >= 3 && id[0] == '<' && id[len(id)-1] == '>' &&
		len(id) <= 250 && isPrintableASCIISlice(id[1:len(id)-1], '>')
}

func reservedMessageID(id string) bool {
	return id == "<0>" /* {RFC 977} */ ||
		id == "<keepalive@dummy.tld>" /* srndv2 */
}

func validMessageNum(n uint64) bool {
	return int64(n) > 0
}

func validGroupSlice(s []byte) bool {
	for _, c := range s {
		if !((c >= 0x22 && c <= 0x29) || c == 0x2B ||
			(c >= 0x2D && c <= 0x3E) || (c >= 0x40 && c <= 0x5A) ||
			(c >= 0x5E && c <= 0x7E) || c >= 0x80) {
			return false
		}
	}
	return len(s) != 0
}

func commonArticleHandler(c *ConnState, kind int, args [][]byte) {
	if len(args) > 0 {
		id := args[0]
		sid := unsafeBytesToStr(id)
		num, e := strconv.ParseUint(sid, 10, 64)
		if e != nil {
			// either non-number or too huge number.
			// ParseUint does not verify rest of string if it's too huge, so treat as invalid.

			// check if Message-ID
			if !validMessageID(id) {
				c.w.PrintfLine("501 unrecognised message identifier")
				return
			}

			if reservedMessageID(sid) || !setA[kind].byMsgID(c, id[1:len(id)-1]) {
				c.w.ResNoArticleWithThatMsgID()
			}
			return
		}

		if c.CurrentGroup == nil {
			c.w.ResNoNewsgroupSelected()
			return
		}

		if validMessageNum(num) || !setA[kind].byNum(c, num) {
			c.w.ResNoArticleWithThatNum()
		}
	} else {
		if c.CurrentGroup == nil {
			c.w.ResNoNewsgroupSelected()
			return
		}

		if !setA[kind].byCurr(c) {
			c.w.ResCurrentArticleNumberIsInvalid()
		}
	}
}

func cmdGroup(c *ConnState, args [][]byte, rest []byte) bool {
	if !validGroupSlice(args[0]) {
		c.w.PrintfLine("501 invalid group name")
		return true
	}
	if !c.prov.SelectGroup(c.w, c, args[0]) {
		c.w.ResNoSuchNewsgroup()
	}
	return true
}

func cmdListGroup(c *ConnState, args [][]byte, rest []byte) bool {
	var group []byte
	if len(args) > 0 {
		if !validGroupSlice(args[0]) {
			c.w.PrintfLine("501 invalid group name")
			return true
		}
		group = args[0]
	} else {
		if c.CurrentGroup == nil {
			c.w.ResNoNewsgroupSelected()
			return true
		}
	}

	rmin := int64(1)
	rmax := int64(-1)
	if len(args) > 1 {
		srange := unsafeBytesToStr(args[1])
		// [num[-[num]]]
		if i := strings.IndexByte(srange, '-'); i >= 0 {
			if i != 0 {
				n, e := strconv.ParseUint(srange[:i], 10, 64)
				if e != nil {
					c.w.PrintfLine("501 invalid range")
					return true
				}
				if int64(n) >= 0 {
					rmin = int64(n)
				}
			}
			if i+1 != len(srange) {
				n, e := strconv.ParseUint(srange[i+1:], 10, 64)
				if e != nil {
					c.w.PrintfLine("501 invalid range")
					return true
				}
				if int64(n) >= 0 {
					rmax = int64(n)
				}
			}
		} else {
			n, e := strconv.ParseUint(srange, 10, 64)
			if e != nil {
				c.w.PrintfLine("501 invalid range")
				return true
			}
			rmin = int64(n)
			rmax = rmin
		}
	}

	if !c.prov.SelectAndListGroup(c.w, c, group, rmin, rmax) {
		c.w.ResNoSuchNewsgroup()
	}
	return true
}

func cmdNext(c *ConnState, args [][]byte, rest []byte) bool {
	if c.CurrentGroup == nil {
		c.w.ResNoNewsgroupSelected()
		return true
	}
	c.prov.SelectNextArticle(c.w, c)
	return true
}

func cmdLast(c *ConnState, args [][]byte, rest []byte) bool {
	if c.CurrentGroup == nil {
		c.w.ResNoNewsgroupSelected()
		return true
	}
	c.prov.SelectPrevArticle(c.w, c)
	return true
}

func cmdNewGroups(c *ConnState, args [][]byte, rest []byte) bool {
	// TODO
	return true
}

func listCmdActive(c *ConnState, args [][]byte, rest []byte) bool {
	// TODO
	return true
}

func listCmdNewsgroups(c *ConnState, args [][]byte, rest []byte) bool {
	// TODO
	return true
}

func listCmdOverviewFmt(c *ConnState, args [][]byte, rest []byte) bool {
	// TODO
	return true
}
