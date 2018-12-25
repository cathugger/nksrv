package psqlib

import (
	"centpd/lib/sqlbucket"
	"fmt"
	"sync"
)

const (
	st_NNTP_articleNumByMsgID  = iota
	st_NNTP_articleMsgIDByNum

	st_NNTP_articleGetByGPID

	st_NNTP_SelectGroup
	st_NNTP_SelectAndListGroup

	st_NNTP_SelectNextArticle
	st_NNTP_SelectPrevArticle

	st_NNTP_ListNewNews_all
	st_NNTP_ListNewNews_one
	st_NNTP_ListNewNews_all_group

	st_NNTP_ListNewGroups

	st_NNTP_ListActiveGroups_all
	st_NNTP_ListActiveGroups_one

	st_NNTP_GetOverByMsgID
	st_NNTP_GetOverByRange
	st_NNTP_GetOverByCurr

	st_NNTP_GetHdrByMsgID_msgid
	st_NNTP_GetHdrByMsgID_subject
	st_NNTP_GetHdrByMsgID_any
	st_NNTP_GetHdrByRange_msgid
	st_NNTP_GetHdrByRange_subject
	st_NNTP_GetHdrByRange_any
	st_NNTP_GetHdrByCurr_msgid
	st_NNTP_GetHdrByCurr_subject
	st_NNTP_GetHdrByCurr_any

	st_max
)

var st_list [st_max]string
var st_loaderr error

var st_names = [st_max]string{
	"nntp_article_num_by_msgid",
	"nntp_article_msgid_by_num",

	"nntp_article_get_gpid",

	"nntp_select",
	"nntp_select_and_list",

	"nntp_next",
	"nntp_last",

	"nntp_newnews_all",
	"nntp_newnews_one",
	"nntp_newnews_all_group",

	"nntp_newgroups",

	"nntp_listactive_all",
	"nntp_listactive_one",

	"nntp_over_msgid",
	"nntp_over_range",
	"nntp_over_curr",

	"nntp_hdr_msgid_msgid",
	"nntp_hdr_msgid_subject",
	"nntp_hdr_msgid_any",
	"nntp_hdr_range_msgid",
	"nntp_hdr_range_subject",
	"nntp_hdr_range_any",
	"nntp_hdr_curr_msgid",
	"nntp_hdr_curr_subject",
	"nntp_hdr_curr_any",
}

func loadStatements() {
	var err error

	const fn = "aux/psqlib/nntp.sql"
	stmts, err := sqlbucket.LoadFromFile(fn)
	if err != nil {
		st_loaderr = fmt.Errorf("err loading %s: %v", fn, err)
		return
	}

	for i := range st_names {
		sn := st_names[i]
		sl := stmts[sn]
		if len(sl) != 1 {
			st_loaderr = fmt.Errorf(
				"wrong count %d for statement %s", len(sl), sn)
			return
		}
		st_list[i] = sl[0]
	}
}

var st_once sync.Once
