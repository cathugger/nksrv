package psqlib

import (
	au "nksrv/lib/asciiutils"
	. "nksrv/lib/logx"
)

func isSubjectVoid(x string) bool {
	// ignore () if any; pointless, probably won't happen in practice
	if len(x) != 0 && x[0] == '(' && x[len(x)-1] == ')' {
		x = au.TrimWSString(x[1 : len(x)-1])
	}
	// content-less subjects some shitty nodes like spamming
	return x == "" ||
		au.EqualFoldString(x, "None") ||
		au.EqualFoldString(x, "no subject")
}

func isSubjectEmpty(s string, isReply, isSage bool, ref_subject string) bool {
	if isSubjectVoid(s) {
		return true
	}
	if isReply {
		// content-less copy of parent subject
		// XXX do we actually need this check?
		if au.EqualFoldString(s, ref_subject) {
			return true
		}
		// sage subject of x-sage message
		if isSage && au.EqualFoldString(s, "sage") {
			return true
		}
		// if after above checks it doesn't start with Re: it's legit
		if !au.StartsWithFoldString(s, "Re:") {
			return false
		}

		// too much newsreaders doing Re: for posts they directly answer
		// so consider everything starting with Re: empty regardless of content
		return true
	} else {
		return false
	}
}
