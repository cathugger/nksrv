package nntp

// 2** - successful completion

func (r Responder) ResGroupSuccessfullySelected(
	est, lo, hi uint64, group string) error {

	return r.PrintfLine("211 %d %d %d %s", est, lo, hi, group)
}

func (r Responder) ResArticleNumbersFollow(
	est, lo, hi uint64, group string) error {

	return r.PrintfLine("211 %d %d %d %s list follows", est, lo, hi, group)
}

func (r Responder) ResListFollows() error {
	return r.PrintfLine("215 list follows")
}

func (r Responder) ResArticleFollows(num uint64, msgid CoreMsgIDStr) error {
	return r.PrintfLine("220 %d <%s> cominngg!!!", num, msgid)
}

func (r Responder) ResHeadFollows(num uint64, msgid CoreMsgIDStr) error {
	return r.PrintfLine("221 %d <%s> head incoming", num, msgid)
}

func (r Responder) ResXHdrFollow() error {
	return r.PrintfLine("221 headers follow")
}

func (r Responder) ResBodyFollows(num uint64, msgid CoreMsgIDStr) error {
	return r.PrintfLine("222 %d <%s> body is coming", num, msgid)
}

func (r Responder) ResArticleFound(num uint64, msgid CoreMsgIDStr) error {
	return r.PrintfLine("223 %d <%s> it's here", num, msgid)
}

func (r Responder) ResOverviewInformationFollows() error {
	return r.PrintfLine("224 ovewview info follows")
}

func (r Responder) ResHdrFollow() error {
	return r.PrintfLine("225 headers follow")
}

func (r Responder) ResListOfNewArticlesFollows() error {
	return r.PrintfLine("230 new articles list")
}

func (r Responder) ResListOfNewNewsgroupsFollows() error {
	return r.PrintfLine("231 new newsgroups list")
}

func (r Responder) ResTransferSuccess() error {
	return r.PrintfLine("235 got it :>")
}

func (r Responder) ResArticleWanted(msgid CoreMsgID) error {
	return r.PrintfLine("238 <%s>", msgid)
}

func (r Responder) ResArticleTransferedOK(msgid CoreMsgID) error {
	return r.PrintfLine("239 <%s>", msgid)
}

func (r Responder) ResPostingAccepted() error {
	return r.PrintfLine("240 article taken in")
}

func (r Responder) ResXListening() error {
	return r.PrintfLine("290 listening in")
}

func (r Responder) ResXWaitAwake() error {
	return r.PrintfLine("291 ohayo~")
}

// 3** - successful continuation

func (r Responder) ResSendArticleToBeTransferred() error {
	return r.PrintfLine("335 yes want")
}

func (r Responder) ResSendArticleToBePosted() error {
	return r.PrintfLine("340 I consent")
}

// 4** - temporary errors

func (r Responder) ResInternalError(e error) error {
	if e != nil {
		return r.PrintfLine("403 internal error: %v", e)
	} else {
		return r.PrintfLine("403 internal error")
	}
}

func (r Responder) ResNoSuchNewsgroup() error {
	return r.PrintfLine("411 I don't see any such newsgroup")
}

func (r Responder) ResNoNewsgroupSelected() error {
	return r.PrintfLine("412 no newsgroup selected fam")
}

func (r Responder) ResCurrentArticleNumberIsInvalid() error {
	return r.PrintfLine("420 current blaze article number isn't valid")
}

func (r Responder) ResXNoArticles() error {
	return r.PrintfLine("420 no articles")
}

func (r Responder) ResNoNextArticleInThisGroup() error {
	return r.PrintfLine("421 no next article")
}

func (r Responder) ResNoPrevArticleInThisGroup() error {
	return r.PrintfLine("422 no prev article")
}

func (r Responder) ResNoArticleWithThatNum() error {
	return r.PrintfLine("423 no article with that number")
}

func (r Responder) ResNoArticlesInThatRange() error {
	return r.PrintfLine("423 no articles in that range")
}

func (r Responder) ResNoArticleWithThatMsgID() error {
	return r.PrintfLine("430 no article with that Message-ID")
}

func (r Responder) ResArticleWantLater(msgid CoreMsgID) error {
	return r.PrintfLine("431 <%s>", msgid)
}

func (r Responder) ResTransferNotWanted() error {
	return r.PrintfLine("435 n-no")
}

func (r Responder) ResTransferFailed() error {
	// failed for whatever reason, can resend
	return r.PrintfLine("436 transfer failed, plz resend later")
}

func (r Responder) ResTransferRejected(e error) error {
	// article not wanted, don't resend
	if e == nil {
		return r.PrintfLine("437 transfer rejected, don't wanna")
	} else {
		return r.PrintfLine("437 transfer rejected, don't wanna (%v)", e)
	}
}

func (r Responder) ResArticleNotWanted(msgid CoreMsgID) error {
	return r.PrintfLine("438 <%s>", msgid)
}

func (r Responder) ResArticleRejected(msgid CoreMsgID, err error) error {
	if err == nil {
		return r.PrintfLine("439 <%s>", msgid)
	} else {
		return r.PrintfLine("439 <%s> %v", msgid, err)
	}
}

func (r Responder) ResPostingNotPermitted() error {
	return r.PrintfLine("440 article injection isn't allowed")
}

func (r Responder) ResPostingFailed(e error) error {
	if e == nil {
		return r.PrintfLine("441 article injection failed")
	} else {
		return r.PrintfLine("441 article injection failed: %v", e)
	}
}

func (r Responder) ResAuthRequired() error {
	return r.PrintfLine("480 authentication required")
}

func (r Responder) ResXWaitTimeout() error {
	return r.PrintfLine("490 timeout")
}

func (r Responder) ResXWaitCancel() error {
	return r.PrintfLine("491 cancelled")
}

// 5** - pernament errors

func (r Responder) ResBadMessageID() error {
	return r.PrintfLine("501 invalid Message-ID")
}

func (r Responder) ResXPermission() error {
	// {RFC 977} 502 access restriction or permission denied
	// {RFC 3977} Meaning for all other commands: command not permitted (and there is no way for the client to change this).
	return r.PrintfLine("502 no permission")
}
