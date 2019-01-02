package mail

import (
	mm "centpd/lib/minimail"
)

func NextValidReference(s string) (mm.FullMsgIDStr, string) {
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

		x := mm.FullMsgIDStr(s[p:i])
		if mm.ValidMessageIDStr(x) {
			return x, s[i:]
		}
	}
	return "", ""
}

func ExtractFirstValidReference(s string) (ref mm.FullMsgIDStr) {
	ref, _ = NextValidReference(s)
	return
}

func ExtractAllValidReferences(
	refs []mm.FullMsgIDStr, s string) []mm.FullMsgIDStr {

	for {
		var x mm.FullMsgIDStr
		x, s = NextValidReference(s)
		if x != "" {
			refs = append(refs, x)
		}
		if s == "" {
			break
		}
	}
	return refs
}
