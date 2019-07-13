package nntp

import (
	"crypto/tls"
	"fmt"
	"sort"
	"time"

	. "nksrv/lib/logx"
)

type (
	commandMapType  map[string]*command
	commandListType []string
)

var (
	commandMap  commandMapType
	commandList commandListType

	listCommandMap  commandMapType
	listCommandList commandListType

	authCommandMap  commandMapType
	authCommandList commandListType
)

func init() {
	sortedCmdMapSlice := func(m commandMapType) (l commandListType) {
		l = make(commandListType, len(m))
		i := 0
		for k := range m {
			l[i] = k
			i++
		}
		sort.Strings(l)
		return
	}

	// maps are initialized there because go is fucking stupid
	// and somehow treats cmdHelp as causing reference cycle

	commandMap = commandMapType{
		"": &command{
			cmdfunc:    cmdVoid,
			allowextra: true,
		},
		"CAPABILITIES": &command{
			cmdfunc:    cmdCapabilities,
			allowextra: true,
			help:       "- print server's capabilities.",
		},
		"MODE": &command{
			cmdfunc: cmdMode,
			minargs: 1,
			maxargs: 1,
		},
		"HELP": &command{
			cmdfunc: cmdHelp,
			maxargs: 2,
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
		"AUTHINFO": &command{
			cmdfunc:    cmdAuthInfo,
			allowextra: true,
		},
		"STARTTLS": &command{
			cmdfunc: cmdStartTLS,
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
		"NEWNEWS": &command{
			cmdfunc: cmdNewNews,
			minargs: 3,
			maxargs: 5, // <distributions> {RFC 977}
			help:    "wildmat [YY]YYMMDD hhmmss [GMT] - list newsgroups created since specified date.",
		},

		"XGTITLE": &command{
			cmdfunc: cmdXGTitle,
			maxargs: 1,
			help:    "[wildmat] - same as LIST NEWSGROUPS.",
		},

		"OVER": &command{
			cmdfunc: func(c *ConnState, args [][]byte, rest []byte) bool {
				commonCmdOver(c, args, true)
				return true
			},
			maxargs: 1,
			help:    "[range|<message-id>] - query overview of article(s) specified by range or Message-ID, or currently selected article.",
		},
		"XOVER": &command{
			cmdfunc: func(c *ConnState, args [][]byte, rest []byte) bool {
				commonCmdOver(c, args, false)
				return true
			},
			maxargs: 1,
			help:    "- same as OVER.",
		},
		"HDR": &command{
			cmdfunc: func(c *ConnState, args [][]byte, rest []byte) bool {
				commonCmdHdr(c, args, true)
				return true
			},
			minargs: 1,
			maxargs: 2,
			help:    "field [range|<message-id>] - query header field of article(s).",
		},
		"XHDR": &command{
			cmdfunc: func(c *ConnState, args [][]byte, rest []byte) bool {
				commonCmdHdr(c, args, false)
				return true
			},
			minargs: 1,
			maxargs: 2,
			help:    "field [range|<message-id>] - query header field of article(s).",
		},

		"POST": &command{
			cmdfunc: cmdPost,
			help:    "- perform article posting.",
		},
		"IHAVE": &command{
			cmdfunc: cmdIHave,
			minargs: 1,
			maxargs: 1,
			help:    "<message-id> - offer and perform article transfer.",
		},
		"CHECK": &command{
			cmdfunc: cmdCheck,
			minargs: 1,
			maxargs: 1,
			help:    "<message-id> - check if server desires article.",
		},
		"TAKETHIS": &command{
			cmdfunc: cmdTakeThis,
			minargs: 1,
			maxargs: 1,
			help:    "<message-id> - It's dangerous to go alone! Take this.",
		},
	}
	commandList = sortedCmdMapSlice(commandMap)

	listCommandMap = commandMapType{
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
			help:    "- list metadata fields returned by OVER command.",
		},
	}
	listCommandList = sortedCmdMapSlice(listCommandMap)

	authCommandMap = commandMapType{
		"USER": &command{
			cmdfunc: authCmdUser,
			minargs: 1,
			maxargs: 1,
			help:    "username - specify user name.",
		},
		"PASS": &command{
			cmdfunc: authCmdPass,
			minargs: 1,
			maxargs: 1,
			help:    "password - specify password.",
		},
		"SASL": &command{
			cmdfunc: authCmdSASL,
			minargs: 1,
			maxargs: 2,
			help:    "mechanism [initial-response] - utilize Simple Authentication and Security Layer.",
		},
	}
	authCommandList = sortedCmdMapSlice(authCommandMap)
}

func cmdVoid(c *ConnState, args [][]byte, rest []byte) bool {
	if len(rest) != 0 {
		AbortOnErr(c.w.PrintfLine("501 command must not start with space"))
	}
	// otherwise ignore
	return true
}

func cmdCapabilities(c *ConnState, args [][]byte, rest []byte) bool {
	AbortOnErr(c.w.PrintfLine("101 capability list follows"))

	rCfg := c.srv.GetRunCfg()

	dw := c.w.DotWriter()

	fmt.Fprintf(dw, "VERSION 2\n")

	if c.advertisePlaintextAuth(rCfg) {
		fmt.Fprintf(dw, "AUTHINFO USER\n")
	}

	if c.AllowReading {
		fmt.Fprintf(dw, "READER\n")
	}

	if c.AllowPosting {
		if c.prov.SupportsPost() {
			fmt.Fprintf(dw, "POST\n")
		}
		if c.prov.SupportsIHave() {
			fmt.Fprintf(dw, "IHAVE\n")
		}
		if c.prov.SupportsStream() {
			fmt.Fprintf(dw, "STREAMING\n")
		}
	}

	if c.AllowReading {
		if c.prov.SupportsNewNews() {
			fmt.Fprintf(dw, "NEWNEWS\n")
		}

		if !c.prov.SupportsOverByMsgID() {
			fmt.Fprintf(dw, "OVER\n")
		} else {
			fmt.Fprintf(dw, "OVER MSGID\n")
		}

		if c.prov.SupportsHdr() {
			fmt.Fprintf(dw, "HDR\n")
		}
	}

	if c.AllowReading || c.AllowPosting {
		fmt.Fprintf(dw, "LIST ACTIVE NEWSGROUPS OVERVIEW.FMT\n")
	}

	if rCfg.TLSConfig != nil && !c.tlsStarted() {
		fmt.Fprintf(dw, "STARTTLS\n")
	}

	// XXX maybe include backend identification
	fmt.Fprintf(dw, "IMPLEMENTATION CNTPD\n")

	AbortOnErr(dw.Close())

	return true
}

func cmdMode(c *ConnState, args [][]byte, rest []byte) bool {
	mode := args[0]
	ToUpperASCII(mode)
	smode := unsafeBytesToStr(mode)

	if smode == "READER" {
		if !c.AllowReading {
			AbortOnErr(c.w.ResAuthRequired())
			return true
		}
		if c.AllowPosting {
			AbortOnErr(c.w.PrintfLine("200 posting allowed"))
		} else {
			AbortOnErr(c.w.PrintfLine("201 posting forbidden"))
		}
	} else if smode == "STREAM" {
		if !c.prov.SupportsStream() {
			AbortOnErr(c.w.PrintfLine("503 STREAMING unimplemented"))
			return true
		}
		if c.AllowPosting {
			AbortOnErr(c.w.PrintfLine("203 streaming permitted"))
		} else {
			AbortOnErr(c.w.ResAuthRequired())
		}
	} else {
		AbortOnErr(c.w.PrintfLine("503 requested MODE not supported"))
	}
	return true
}

func cmdHelp(c *ConnState, args [][]byte, rest []byte) bool {
	AbortOnErr(c.w.PrintfLine("100 here's manual"))

	dw := c.w.DotWriter()
	defer dw.Close()

	if len(args) != 0 {
		ToUpperASCII(args[0])
		k := unsafeBytesToStr(args[0])

		noSuchCommand := func() { fmt.Fprintf(dw, "no such command\n") }
		noInfoAboutThisCommand := func() {
			fmt.Fprintf(dw, "no info about this command\n")
		}

		cmd, ok := commandMap[k]
		if !ok {
			noSuchCommand()
			return true
		}

		if len(args) != 1 {
			printCmd := func(p string, m commandMapType, k string) {
				c, ok := m[k]
				if !ok {
					noSuchCommand()
					return
				}
				if c.help == "" {
					noInfoAboutThisCommand()
					return
				}
				fmt.Fprintf(dw, "%s %s %s\n", p, k, c.help)
			}
			ToUpperASCII(args[1])
			ck := unsafeBytesToStr(args[1])
			if k == "LIST" {
				printCmd(k, listCommandMap, ck)
			} else if k == "AUTHINFO" {
				printCmd(k, listCommandMap, ck)
			} else {
				noSuchCommand()
			}
			return true
		}

		if cmd.help == "" {
			noInfoAboutThisCommand()
			return true
		}

		fmt.Fprintf(dw, "%s %s\n", k, cmd.help)
		return true
	}

	printCmds := func(p string, l commandListType, m commandMapType) {
		for _, k := range l {
			c := m[k]
			if c.help != "" {
				fmt.Fprintf(dw, "%s %s %s\n", p, k, c.help)
			}
		}
	}
	for _, k := range commandList {
		cmd := commandMap[k]
		if cmd.help != "" {
			fmt.Fprintf(dw, "%s %s\n", k, cmd.help)
		}
		if k == "LIST" {
			printCmds(k, listCommandList, listCommandMap)
		} else if k == "AUTHINFO" {
			printCmds(k, authCommandList, authCommandMap)
		}
	}

	return true
}

func cmdList(c *ConnState, args [][]byte, rest []byte) bool {
	args = args[:0] // reuse

	if len(rest) == 0 {
		listCmdActive(c, args, nil)
		return true
	}

	x := parseKeyword(rest)

	cmd, ok := listCommandMap[string(rest[:x])]
	if !ok {
		AbortOnErr(c.w.PrintfLine("501 unrecognised LIST keyword"))
		return true
	}

	if x >= len(rest) {
		goto argsparsed
	}

	for {
		// skip spaces
		for {
			x++
			if x >= len(rest) {
				goto argsparsed
			}
			if rest[x] != ' ' && rest[x] != '\t' {
				break
			}
		}
		if len(args) >= cmd.maxargs {
			if !cmd.allowextra {
				AbortOnErr(c.w.PrintfLine("501 too much parameters"))
			} else {
				cmd.cmdfunc(c, args, rest[x:])
			}
			return true
		}
		sx := x
		// skip non-spaces
		for {
			x++
			if x >= len(rest) {
				args = append(args, rest[sx:x])
				goto argsparsed
			}
			if rest[x] == ' ' || rest[x] == '\t' {
				args = append(args, rest[sx:x])
				break
			}
		}
	}
argsparsed:
	if len(args) < cmd.minargs {
		AbortOnErr(c.w.PrintfLine("501 not enough parameters"))
		return true
	}
	cmd.cmdfunc(c, args, nil)
	return true
}

func cmdQuit(c *ConnState, args [][]byte, rest []byte) bool {
	_ = c.w.PrintfLine("205 goodbye.")
	// will close gracefuly
	return false
}

func cmdDate(c *ConnState, args [][]byte, rest []byte) bool {
	t := time.Now().UTC()
	Y, M, D := t.Date()
	h, m, s := t.Clock()
	// 111 YYYYMMDDhhmmss    Server date and time
	// XXX will break format when year>9999
	AbortOnErr(c.w.PrintfLine(
		"111 %4d%2d%2d%2d%2d%2d YYYYMMDDhhmmss", Y, M, D, h, m, s))
	return true
}

func cmdSlave(c *ConnState, args [][]byte, rest []byte) bool {
	AbortOnErr(c.w.PrintfLine("202 slave status noted")) // :^)
	return true
}

func cmdStartTLS(c *ConnState, args [][]byte, rest []byte) bool {
	rcfg := c.srv.GetRunCfg()
	if rcfg.TLSConfig == nil {
		AbortOnErr(c.w.PrintfLine("580 TLS not configured"))
		return true
	}
	if c.tlsStarted() {
		AbortOnErr(c.w.PrintfLine("502 TLS already activated"))
		return true
	}

	AbortOnErr(c.w.PrintfLine("382 continue with TLS negotiation"))
	tlsc := tls.Client(c.conn, rcfg.TLSConfig)
	err := tlsc.Handshake()
	if err != nil {
		c.log.LogPrintf(WARN, "STARTTLS TLS negotiation error: %v", err)
		return false
	}
	c.r.SetReader(tlsc)
	c.w.Writer.W.Reset(tlsc)
	c.Cleanup()
	c.postTLS(rcfg, tlsc)

	return true
}
