package psqlib

import (
	. "nksrv/lib/logx"
)

type headerRestriction struct {
	h string // header
	o bool   // optional (allow absence?)
}

var hdrNNTPPostRestrict = [...]headerRestriction{
	// NetNews stuff specified in {RFC 5536}
	{"Message-ID", true},
	{"From", false},
	{"Date", true},
	{"Newsgroups", false},
	{"Path", true},
	{"Subject", true}, // more lax than {RFC 5536} (no subject is much better than "none")

	// {RFC 5322}
	{"Sender", true},
	{"Reply-To", true},
	{"To", true},
	{"Cc", true},
	{"Bcc", true},
	{"In-Reply-To", true},
	{"References", true},

	// some extras we process
	{"Injection-Date", true},
	{"NNTP-Posting-Date", true},
}

// mandatory headers for transmission. POST uses separate system
var hdrNNTPTransferRestrict = [...]headerRestriction{
	// NetNews stuff specified in {RFC 5536}
	{"Message-ID", true}, // special handling
	{"From", true},       // idfk why there are articles like this
	{"Date", false},
	{"Newsgroups", false},
	{"Path", false},
	{"Subject", true}, // more lax than {RFC 5536} (no subject is much better than "none")

	// {RFC 5322}
	{"Sender", true},
	{"Reply-To", true},
	{"To", true},
	{"Cc", true},
	{"Bcc", true},
	{"In-Reply-To", true},
	{"References", true},

	// some extras we process
	{"Injection-Date", true},
	{"NNTP-Posting-Date", true},
}

const (
	maxNameSize    = 255
	maxSubjectSize = 255
)
