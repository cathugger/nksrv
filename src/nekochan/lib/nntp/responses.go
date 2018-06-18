package nntp

// 2** - successful completion

func (r Responder) ResGroupSuccessfullySelected(est, lo, hi int64, group string) {
	r.PrintfLine("211 %d %d %d %s", est, lo, hi, group)
}

func (r Responder) ResArticleNumbersFollow(est, lo, hi int64, group string) {
	r.PrintfLine("211 %d %d %d %s list follows", est, lo, hi, group)
}

func (r Responder) ResListOfNewsgroupsFollows() {
	r.PrintfLine("215 newsgroups list")
}

func (r Responder) ResArticleFollows(num int64, msgid string) {
	r.PrintfLine("220 %d %s cominngg!!!", num, msgid)
}

func (r Responder) ResHeadFollows(num int64, msgid string) {
	r.PrintfLine("221 %d %s head incoming", num, msgid)
}

func (r Responder) ResBodyFollows(num int64, msgid string) {
	r.PrintfLine("222 %d %s body is coming", num, msgid)
}

func (r Responder) ResArticleFound(num int64, msgid string) {
	r.PrintfLine("223 %d %s it's here", num, msgid)
}

func (r Responder) ResListOfNewNewsgroupsFollows() {
	r.PrintfLine("231 new newsgroups list")
}

func (r Responder) ResTransferSuccess() {
	r.PrintfLine("235 got it :>")
}

func (r Responder) ResPostingAccepted() {
	r.PrintfLine("240 article taken in")
}

// 3** - successful continuation

func (r Responder) ResSendArticleToBeTransferred(msgid string) {
	r.PrintfLine("335 yes want")
}

func (r Responder) ResSendArticleToBePosted() {
	r.PrintfLine("340 I consent")
}

// 4** - temporary errors

func (r Responder) ResNoSuchNewsgroup() {
	r.PrintfLine("411 I don't see any such newsgroup")
}

func (r Responder) ResNoNewsgroupSelected() {
	r.PrintfLine("412 no newsgroup selected fam")
}

func (r Responder) ResCurrentArticleNumberIsInvalid() {
	r.PrintfLine("420 current blaze article number isn't valid")
}

func (r Responder) ResNoNextArticleInThisGroup() {
	r.PrintfLine("421 no next article")
}

func (r Responder) ResNoPrevArticleInThisGroup() {
	r.PrintfLine("422 no prev article")
}

func (r Responder) ResNoArticleWithThatNum() {
	r.PrintfLine("423 no article with that number")
}

func (r Responder) ResNoArticlesInThatRange() {
	r.PrintfLine("423 no articles in that range")
}

func (r Responder) ResNoArticleWithThatMsgID() {
	r.PrintfLine("430 no article with that Message-ID")
}

func (r Responder) ResArticleNotWanted() {
	r.PrintfLine("435 n-no")
}

func (r Responder) ResTransferFailed() {
	r.PrintfLine("436 transfer failed")
}

func (r Responder) ResTransferRejected() {
	r.PrintfLine("437 transfer rejected, don't wanna")
}

func (r Responder) ResPostingNotPermitted() {
	r.PrintfLine("440 article injection isn't allowed")
}

func (r Responder) ResPostingFailed() {
	r.PrintfLine("441 article injection failed")
}

func (r Responder) ResAuthRequired() {
	r.PrintfLine("480 authentication required")
}

// 5** - pernament errors

func (r Responder) ResBadMessageID() {
	r.PrintfLine("501 invalid Message-ID")
}
