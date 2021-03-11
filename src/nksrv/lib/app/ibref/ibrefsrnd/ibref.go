package ibrefsrnd

import (
	"regexp"
	"sort"
	"strings"
)

var (
	// normal reference. usually within current board
	reRef = regexp.MustCompile(
		`>> ?([0-9a-fA-F]{8,40})\b`)
	// cross reference. explicitly specifies board
	reCRef = regexp.MustCompile(
		`>>> ?/([0-9a-zA-Z+_.-]{1,255})/(?: ?([0-9a-fA-F]{8,40})\b)?`)
)

// syntax of RFC 5536 seems restrictive enough to not allow much false positives
const reAtom = "[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+"
const reDAtom = reAtom + "(?:\\." + reAtom + ")*"
const reMDText = "[\x21-\x3D\x3F-\x5A\x5E-\x7E]"
const reNoFoldLit = "\\[" + reMDText + "*\\]"

// Message-ID reference
var reMsgID = regexp.MustCompile(
	"<" + reDAtom + "@(?:" + reDAtom + "|" + reNoFoldLit + ")>")

type Reference struct {
	Board string
	Post  string
	MsgID string
}

type Index struct {
	Start int
	End   int
}

type tiedSorter struct {
	srefs []Reference
	irefs []Index
}

func (s tiedSorter) Len() int {
	return len(s.srefs)
}

func (s tiedSorter) Less(i, j int) bool {
	return s.irefs[i].Start < s.irefs[j].Start
}

func (s tiedSorter) Swap(i, j int) {
	s.srefs[i], s.srefs[j] = s.srefs[j], s.srefs[i]
	s.irefs[i], s.irefs[j] = s.irefs[j], s.irefs[i]
}

func ParseReferences(msg string) (srefs []Reference, irefs []Index) {
	var sm [][]int

	sm = reRef.FindAllStringSubmatchIndex(msg, -1)
	for i := range sm {
		srefs = append(srefs, Reference{
			Post: strings.ToLower(msg[sm[i][2]:sm[i][3]]),
		})
		irefs = append(irefs, Index{
			Start: sm[i][0],
			End:   sm[i][1],
		})
	}

	sm = reCRef.FindAllStringSubmatchIndex(msg, -1)
	for i := range sm {
		x := Reference{
			Board: msg[sm[i][2]:sm[i][3]],
		}
		if sm[i][4] >= 0 {
			x.Post = strings.ToLower(msg[sm[i][4]:sm[i][5]])
		}
		srefs = append(srefs, x)

		irefs = append(irefs, Index{
			Start: sm[i][0],
			End:   sm[i][1],
		})
	}

	sm = reMsgID.FindAllStringIndex(msg, -1)
	for i := range sm {
		if sm[i][1]-sm[i][0] > 250 || sm[i][1]-sm[i][0] < 3 {
			continue
		}
		srefs = append(srefs, Reference{
			MsgID: msg[sm[i][0]+1 : sm[i][1]-1],
		})
		irefs = append(irefs, Index{
			Start: sm[i][0],
			End:   sm[i][1],
		})
	}

	// sort by position
	sort.Sort(tiedSorter{srefs: srefs, irefs: irefs})

	// remove overlaps, if any
	for i := 1; i < len(irefs); i++ {
		if irefs[i-1].End > irefs[i].Start {
			srefs = append(srefs[:i], srefs[i+1:]...)
			irefs = append(irefs[:i], irefs[i+1:]...)
			i--
		}
	}

	// limit
	if len(srefs) > 255 {
		srefs = srefs[:255]
		irefs = irefs[:255]
	}

	return
}
