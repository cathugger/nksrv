package ibref_nntp

import (
	"regexp"
	"sort"
	"strings"
)

var re_ref = regexp.MustCompile(
	`>> ?([0-9a-fA-F]{8,40})\b`)
var re_cref = regexp.MustCompile(
	`>>> ?/([0-9a-zA-Z+_.-]{1,255})/(?: ?([0-9a-fA-F]{8,40})\b)?`)

// syntax of RFC 5536 seems restrictive enough to not allow much false positives
const re_atom = "[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+"
const re_datom = re_atom + "(?:\\." + re_atom + ")*"
const re_mdtext = "[\x21-\x3D\x3F-\x5A\x5E-\x7E]"
const re_nofoldlit = "\\[" + re_mdtext + "*\\]"

var re_msgid = regexp.MustCompile(
	"<" + re_datom + "@(?:" + re_datom + "|" + re_nofoldlit + ")>")

type Reference struct {
	Board string
	Post  string
	MsgID string
}

type IndexReference struct {
	Start int
	End   int

	Reference
}

func ParseReferences(msg string) (srefs []IndexReference) {
	var sm [][]int
	sm = re_ref.FindAllStringSubmatchIndex(msg, -1)
	for i := range sm {
		srefs = append(srefs, IndexReference{
			Start: sm[i][0],
			End:   sm[i][1],

			Reference: Reference{
				Post: strings.ToLower(msg[sm[i][2]:sm[i][3]]),
			},
		})
	}
	sm = re_cref.FindAllStringSubmatchIndex(msg, -1)
	for i := range sm {
		x := IndexReference{
			Start: sm[i][0],
			End:   sm[i][1],

			Reference: Reference{
				Board: msg[sm[i][2]:sm[i][3]],
			},
		}
		if sm[i][4] >= 0 {
			x.Post = strings.ToLower(msg[sm[i][4]:sm[i][5]])
		}
		srefs = append(srefs, x)
	}
	sm = re_msgid.FindAllStringIndex(msg, -1)
	for i := range sm {
		if sm[i][1]-sm[i][0] > 250 || sm[i][1]-sm[i][0] < 3 {
			continue
		}
		x := IndexReference{
			Start: sm[i][0],
			End:   sm[i][1],

			Reference: Reference{
				MsgID: msg[sm[i][0]+1 : sm[i][1]-1],
			},
		}
		srefs = append(srefs, x)
	}
	// sort by position
	sort.Slice(srefs, func(i, j int) bool {
		return srefs[i].Start < srefs[j].Start
	})
	// remove overlaps, if any
	for i := 1; i < len(srefs); i++ {
		if srefs[i-1].End > srefs[i].Start {
			srefs = append(srefs[:i], srefs[i+1:]...)
			i--
		}
	}
	// limit
	if len(srefs) > 255 {
		srefs = srefs[:255]
	}
	return
}
