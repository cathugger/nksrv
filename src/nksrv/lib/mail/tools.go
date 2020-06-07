package mail

import (
	mm "nksrv/lib/minimail"
)

func NextValidReference(s string) (mm.TFullMsgIDStr, string) {
	i := 0
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		if c == '(' {
			i++
			clvl := 1
			q := false
			for i < len(s) {
				cc := s[i]
				i++
				if cc == ')' && !q {
					clvl--
					if clvl <= 0 {
						break
					}
				}
				if cc == '(' && !q {
					clvl++
				}
				q = cc == '\\'
			}
			continue
		}

		p := i
		i++
		for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != '(' {
			i++
		}

		x := mm.TFullMsgIDStr(s[p:i])
		if mm.ValidMessageIDStr(x) {
			return x, s[i:]
		}
	}
	return "", ""
}

func ExtractFirstValidReference(s string) (ref mm.TFullMsgIDStr) {
	ref, _ = NextValidReference(s)
	return
}

func ExtractAllValidReferences(
	refs []string, s string) []string {

	for {
		var x mm.TFullMsgIDStr
		x, s = NextValidReference(s)
		if x != "" {
			refs = append(refs, string(x))
		}
		if s == "" {
			break
		}
	}
	return refs
}
