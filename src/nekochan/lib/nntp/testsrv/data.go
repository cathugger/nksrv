package testsrv

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"nekochan/lib/nntp"
)

var newline = []byte("\n")

type overview struct {
	subject    string
	from       string
	date       string
	msgid      FullMsgIDStr
	references string
	bytes      string
	lines      string
	xref       string
	// not used in NNTP but some things request it so might aswell
	to string
	cc string
}

func (o *overview) GetByHdr(b []byte) (string, bool) {
	nntp.ToLowerASCII(b)
	s := unsafeBytesToStr(b)
	switch s {
	case "subject":
		return o.subject, true
	case "from":
		return o.from, true
	case "date":
		return o.date, true
	case "message-id":
		return string(o.msgid), true
	case "references":
		return o.references, true
	case "bytes", ":bytes":
		return o.bytes, true
	case "lines", ":lines":
		return o.lines, true
	case "xref":
		return o.references, true
	case "to":
		return o.to, true
	case "cc":
		return o.cc, true
	default:
		return "", false
	}
}

type article struct {
	msgid  CoreMsgIDStr
	group  string
	number uint64
	posted time.Time
	over   overview
	head   []byte
	body   []byte
}

type group struct {
	name         string
	info         string
	created      time.Time
	status       byte
	articles     map[uint64]*article
	articlesSort []uint64
}

type server struct {
	name       string
	groups     map[string]*group
	groupsSort []string
	articles   map[CoreMsgIDStr]*article
}

var a1msgid = CoreMsgIDStr("nekosnekosnekosnekos@void.neko.test")
var a1group = "nekos"
var a1num = uint64(9000)

var a1 = article{
	msgid:  a1msgid,
	group:  a1group,
	number: a1num,
	posted: time.Date(2000, 10, 6, 1, 23, 45, 0, time.UTC),
	head: []byte(`Message-ID: <nekosnekosnekosnekos@void.neko.test>
Subject: nekos
Newsgroups: nekos
From: <neko@void.neko.test>
Date: 6 Oct 2000 01:23:45 -0000
Path: void.neko.test
`),
	body: []byte(`Nekos are Love.
Nekos are Life.
`),
}

var g1 = group{
	name:    a1group,
	info:    "group about nekos",
	created: time.Date(2000, 1, 23, 12, 34, 56, 0, time.UTC),
	status:  'y',
	articles: map[uint64]*article{
		a1num: &a1,
	},
}

var s1 = server{
	name: "void.neko.test",
	groups: map[string]*group{
		a1group: &g1,
	},
	articles: map[CoreMsgIDStr]*article{},
}

func sortArticles(g *group) {
	z := g.articlesSort[:0]
	for x := range g.articles {
		z = append(z, x)
	}
	sort.Slice(z, func(i, j int) bool { return z[i] < z[j] })
	g.articlesSort = z
}

func sortGroups(s *server) {
	z := s.groupsSort[:0]
	for x := range s.groups {
		z = append(z, x)
	}
	sort.Strings(z)
	s.groupsSort = z
}

func makeOverview(s *server, a *article) {
	h, _ := nntp.ReadHeaders(bytes.NewReader(a.head), 0)
	numlines := func(b []byte) (n int) {
		for _, c := range b {
			if c == '\n' {
				n++
			}
		}
		return
	}
	a.over.subject = h.H.GetFirst("Subject")
	a.over.from = h.H.GetFirst("From")
	a.over.to = h.H.GetFirst("To")
	a.over.cc = h.H.GetFirst("Cc")
	a.over.date = h.H.GetFirst("Date")
	a.over.msgid = FullMsgIDStr(h.H.GetFirst("Message-ID"))
	a.over.references = h.H.GetFirst("References")
	a.over.bytes = fmt.Sprintf("%d", len(a.body))
	a.over.lines = fmt.Sprintf("%d", numlines(a.body))
	a.over.xref = fmt.Sprintf("%s %s:%d", s.name, a.group, a.number)
}

func prepareServer(s *server) {
	sortGroups(s)
	if s.articles == nil {
		s.articles = make(map[CoreMsgIDStr]*article)
	}
	for _, g := range s1.groups {
		sortArticles(g)
		for _, a := range g.articles {
			s.articles[a.msgid] = a
			makeOverview(s, a)
		}
	}
}

func init() {
	prepareServer(&s1)
}
