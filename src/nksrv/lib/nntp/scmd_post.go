package nntp

func cmdPost(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.prov.SupportsPost() {
		AbortOnErr(c.w.PrintfLine("503 POST unimplemented"))
		return true
	}
	if !c.AllowPosting || !c.prov.HandlePost(c.w, c, c) {
		AbortOnErr(c.w.ResPostingNotPermitted())
	}
	return true
}

func cmdIHave(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.prov.SupportsIHave() {
		AbortOnErr(c.w.PrintfLine("503 IHAVE unimplemented"))
		return true
	}

	id := FullMsgID(args[0])
	if !ValidMessageID(id) {
		AbortOnErr(c.w.ResBadMessageID())
		return true
	}

	if !c.AllowPosting {
		AbortOnErr(c.w.ResAuthRequired())
		return true
	}

	if ReservedMessageID(id) || !c.prov.HandleIHave(c.w, c, c, CutMessageID(id)) {
		AbortOnErr(c.w.ResTransferNotWanted())
	}
	return true
}

func cmdCheck(c *ConnState, args [][]byte, rest []byte) bool {
	if !c.prov.SupportsStream() {
		AbortOnErr(c.w.PrintfLine("503 STREAMING unimplemented"))
		return true
	}

	id := FullMsgID(args[0])
	if !ValidMessageID(id) {
		AbortOnErr(c.w.ResBadMessageID())
		return true
	}

	// check can waste server's resources too
	// but if reading is allowed, then client can do the same in different way
	// so allow it in that case
	if !c.AllowPosting && !c.AllowReading {
		AbortOnErr(c.w.ResAuthRequired())
		return true
	}

	cid := CutMessageID(id)
	if ReservedMessageID(id) || !c.prov.HandleCheck(c.w, c, cid) {
		AbortOnErr(c.w.ResArticleNotWanted(cid))
	}
	return true
}

func cmdTakeThis(c *ConnState, args [][]byte, rest []byte) bool {
	r := c.OpenReader()
	defer r.Discard(-1)

	if !c.prov.SupportsStream() {
		AbortOnErr(c.w.PrintfLine("503 STREAMING unimplemented"))
		return true
	}

	id := FullMsgID(args[0])
	if !ValidMessageID(id) {
		AbortOnErr(c.w.ResBadMessageID())
		return true
	}

	// check can waste server's resources too
	if !c.AllowPosting {
		AbortOnErr(c.w.ResAuthRequired())
		return true
	}

	cid := CutMessageID(id)
	if ReservedMessageID(id) || !c.prov.HandleTakeThis(c.w, c, r, cid) {
		AbortOnErr(c.w.ResArticleRejected(cid, nil))
	}
	return true
}
