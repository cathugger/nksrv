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
	if !ValidMessageID(id) {
		c.w.ResBadMessageID()
		return true
	}

	if !c.AllowPosting {
		c.w.ResAuthRequired()
		return true
	}

	if ReservedMessageID(id) || !c.prov.HandleIHave(c.w, c, c, CutMessageID(id)) {
		c.w.ResTransferNotWanted()
	}
	return true
}

func cmdCheck(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.prov.SupportsStream() {
		c.w.PrintfLine("503 STREAMING unimplemented")
		return true
	}

	id := FullMsgID(args[0])
	if !ValidMessageID(id) {
		c.w.ResBadMessageID()
		return true
	}

	// check can waste server's resources too
	// but if reading is allowed, then client can do the same in different way
	// so allow it in that case
	if !c.AllowPosting && !c.AllowReading {
		c.w.ResAuthRequired()
		return true
	}

	cid := CutMessageID(id)
	if ReservedMessageID(id) || !c.prov.HandleCheck(c.w, c, cid) {
		c.w.ResArticleNotWanted(cid)
	}
	return true
}

func cmdTakeThis(c *ConnState, args [][]byte, rest []byte) bool {
	r := c.OpenReader()
	defer r.Discard(-1)

	if !c.prov.SupportsStream() {
		c.w.PrintfLine("503 STREAMING unimplemented")
		return true
	}

	id := FullMsgID(args[0])
	if !ValidMessageID(id) {
		c.w.ResBadMessageID()
		return true
	}

	// check can waste server's resources too
	if !c.AllowPosting {
		c.w.ResAuthRequired()
		return true
	}

	cid := CutMessageID(id)
	if ReservedMessageID(id) || !c.prov.HandleTakeThis(c.w, c, r, cid) {
		c.w.ResArticleRejected(cid, nil)
	}
	return true
}
