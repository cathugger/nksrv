package mail

import (
	mm "centpd/lib/minimail"
)

func ExtractFirstValidReference(s string) mm.FullMsgIDStr {
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
			return x
		}
	}
	return ""
}
