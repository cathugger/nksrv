package nntp

import "strconv"

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

func commonArticleHandler(c *ConnState, kind int, args [][]byte) {
	if !c.AllowReading {
		c.w.ResAuthRequired()
		return
	}

	if len(args) > 0 {
		id := args[0]

		if ValidMessageID(FullMsgID(id)) {
			mid := FullMsgID(id)
			if ReservedMessageID(mid) || !setA[kind].byMsgID(c, cutMessageID(mid)) {
				c.w.ResNoArticleWithThatMsgID()
			}
			return
		}

		num, e := strconv.ParseUint(unsafeBytesToStr(id), 10, 64)
		if e == nil {
			if c.CurrentGroup == nil {
				c.w.ResNoNewsgroupSelected()
				return
			}

			if validMessageNum(num) || !setA[kind].byNum(c, num) {
				c.w.ResNoArticleWithThatNum()
			}
			return
		}

		c.w.PrintfLine("501 unrecognised message identifier")
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

	if !c.AllowReading {
		c.w.ResAuthRequired()
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
		var valid bool
		if rmin, rmax, valid = parseRange(unsafeBytesToStr(args[1])); !valid {
			c.w.PrintfLine("501 invalid range")
			return true
		}
	}

	if !c.AllowReading {
		c.w.ResAuthRequired()
		return true
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

	// if current group pointer set, reading was allowed already

	c.prov.SelectNextArticle(c.w, c)
	return true
}

func cmdLast(c *ConnState, args [][]byte, rest []byte) bool {
	if c.CurrentGroup == nil {
		c.w.ResNoNewsgroupSelected()
		return true
	}

	// if current group pointer set, reading was allowed already

	c.prov.SelectPrevArticle(c.w, c)
	return true
}

func cmdNewNews(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.prov.SupportsNewNews() {
		c.w.PrintfLine("503 unimplemented")
		return true
	}

	// we use GMT either way so dont even check for that
	// <distributions> is not specified in newest RFC so dont care about that either
	// NEWNEWS wildmat [YY]YYMMDD hhmmss

	wildmat := args[0]
	if !validWildmat(wildmat) {
		c.w.PrintfLine("501 invalid wildmat")
		return true
	}

	qt, valid := parseDateTime(c.w, args[1], args[2])
	if !valid {
		return true
	}

	if !c.AllowReading {
		c.w.ResAuthRequired()
		return true
	}

	c.w.PrintfLine("230 list of new articles follows")
	dw := c.w.DotWriter()
	c.prov.ListNewNews(dw, wildmat, qt)
	dw.Close()

	return true
}

func cmdNewGroups(c *ConnState, args [][]byte, rest []byte) bool {
	// we use GMT either way so dont even check for that
	// <distributions> is not specified in newest RFC so dont care about that either
	// NEWGROUPS [YY]YYMMDD hhmmss
	qt, valid := parseDateTime(c.w, args[0], args[1])
	if !valid {
		return true
	}

	if !c.AllowReading {
		c.w.ResAuthRequired()
		return true
	}

	c.w.PrintfLine("231 list of new groups follows")
	dw := c.w.DotWriter()
	c.prov.ListNewGroups(dw, qt)
	dw.Close()

	return true
}

func cmdOver(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.AllowReading {
		c.w.ResAuthRequired()
		return true
	}

	if len(args) > 0 {
		id := args[0]

		if ValidMessageID(FullMsgID(id)) {
			if !c.prov.SupportsOverByMsgID() {
				c.w.PrintfLine("503 OVER MSGID unimplemented")
				return true
			}
			mid := FullMsgID(id)
			if ReservedMessageID(mid) || !c.prov.GetOverByMsgID(c.w, cutMessageID(mid)) {
				c.w.ResNoArticleWithThatMsgID()
			}
		} else {
			if c.CurrentGroup == nil {
				c.w.ResNoNewsgroupSelected()
				return true
			}

			var rmin, rmax int64
			var valid bool
			if rmin, rmax, valid = parseRange(unsafeBytesToStr(id)); !valid {
				c.w.PrintfLine("501 invalid range")
				return true
			}

			if (rmax >= 0 && rmax < rmin) || !c.prov.GetOverByRange(c.w, c, rmin, rmax) {
				c.w.ResNoArticlesInThatRange()
			}
		}
	} else {
		if c.CurrentGroup == nil {
			c.w.ResNoNewsgroupSelected()
			return true
		}

		if !c.prov.GetOverByCurr(c.w, c) {
			c.w.ResCurrentArticleNumberIsInvalid()
		}
	}
	return true
}
