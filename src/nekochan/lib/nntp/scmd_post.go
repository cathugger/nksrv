package nntp

func cmdPost(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.prov.SupportsPost() {
		c.w.PrintfLine("503 POST unimplemented")
		return true
	}
	if !c.AllowPosting || !c.prov.HandlePost(c.w, c, c) {
		c.w.ResPostingNotPermitted()
	}
	return true
}

func cmdIHave(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.prov.SupportsIHave() {
		c.w.PrintfLine("503 IHAVE unimplemented")
		return true
	}

	id := FullMsgID(args[0])
	if !validMessageID(id) {
		c.w.ResBadMessageID()
		return true
	}

	if !c.AllowPosting {
		c.w.ResAuthRequired()
		return true
	}

	if reservedMessageID(id) || !c.prov.HandleIHave(c.w, c, c, cutMessageID(id)) {
		c.w.ResArticleNotWanted()
	}
	return true
}

func cmdCheck(c *ConnState, args [][]byte, rest []byte) bool {
	id := FullMsgID(args[0])
	if !validMessageID(id) {
		c.w.ResBadMessageID()
		return true
	}

	// check can waste server's resources too
	if !c.AllowPosting {
		c.w.ResAuthRequired()
		return true
	}

	if reservedMessageID(id) || !c.prov.HandleCheck(c.w, c, cutMessageID(id)) {
		c.w.PrintfLine("438 %s", id)
	}
	return true
}

func cmdTakeThis(c *ConnState, args [][]byte, rest []byte) bool {
	r := c.OpenReader()
	defer r.Discard(-1)

	if !c.prov.SupportsIHave() {
		c.w.PrintfLine("503 TAKETHIS unimplemented")
		return true
	}

	id := FullMsgID(args[0])
	if !validMessageID(id) {
		c.w.ResBadMessageID()
		return true
	}

	// check can waste server's resources too
	if !c.AllowPosting {
		c.w.ResAuthRequired()
		return true
	}

	if reservedMessageID(id) || !c.prov.HandleCheck(c.w, c, cutMessageID(id)) {
		c.w.PrintfLine("439 %s", id)
	}
	return true
}
