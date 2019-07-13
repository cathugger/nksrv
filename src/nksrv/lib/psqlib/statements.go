package psqlib

import (
	"fmt"
	"sync"

	"nksrv/lib/sqlbucket"
)

const (
	st_NNTP_articleExistsOrBannedByMsgID = iota
	st_NNTP_articleValidAndBannedByMsgID

	st_NNTP_articleNumByMsgID
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

	st_web_listboards

	st_web_thread_list_page

	st_web_overboard_page

	st_web_thread_catalog

	st_web_thread

	st_web_failref_write
	st_web_failref_find
	st_web_update_post_attrs

	st_web_autoregister_mod
	st_web_delete_by_msgid
	st_web_ban_by_msgid
	st_web_bname_topts_by_tid
	st_web_refresh_bump_by_tid

	st_web_set_mod_priv
	st_web_fetch_and_clear_mod_msgs

	st_max
)

var st_listx [st_max]string
var st_loaderr error

type st_reference struct {
	Bucket string
	Name   string
}

var st_names = [st_max]st_reference{
	st_reference{"nntp", "nntp_article_exists_or_banned_by_msgid"},
	st_reference{"nntp", "nntp_article_valid_and_banned_by_msgid"},

	st_reference{"nntp", "nntp_article_num_by_msgid"},
	st_reference{"nntp", "nntp_article_msgid_by_num"},

	st_reference{"nntp", "nntp_article_get_gpid"},

	st_reference{"nntp", "nntp_select"},
	st_reference{"nntp", "nntp_select_and_list"},

	st_reference{"nntp", "nntp_next"},
	st_reference{"nntp", "nntp_last"},

	st_reference{"nntp", "nntp_newnews_all"},
	st_reference{"nntp", "nntp_newnews_one"},
	st_reference{"nntp", "nntp_newnews_all_group"},

	st_reference{"nntp", "nntp_newgroups"},

	st_reference{"nntp", "nntp_listactive_all"},
	st_reference{"nntp", "nntp_listactive_one"},

	st_reference{"nntp", "nntp_over_msgid"},
	st_reference{"nntp", "nntp_over_range"},
	st_reference{"nntp", "nntp_over_curr"},

	st_reference{"nntp", "nntp_hdr_msgid_msgid"},
	st_reference{"nntp", "nntp_hdr_msgid_subject"},
	st_reference{"nntp", "nntp_hdr_msgid_any"},
	st_reference{"nntp", "nntp_hdr_range_msgid"},
	st_reference{"nntp", "nntp_hdr_range_subject"},
	st_reference{"nntp", "nntp_hdr_range_any"},
	st_reference{"nntp", "nntp_hdr_curr_msgid"},
	st_reference{"nntp", "nntp_hdr_curr_subject"},
	st_reference{"nntp", "nntp_hdr_curr_any"},

	st_reference{"web", "web_listboards"},

	st_reference{"web", "web_thread_list_page"},

	st_reference{"web", "web_overboard_page"},

	st_reference{"web", "web_thread_catalog"},

	st_reference{"web", "web_thread"},

	st_reference{"web", "web_failref_write"},
	st_reference{"web", "web_failref_find"},
	st_reference{"web", "update_post_attrs"},

	st_reference{"web", "autoregister_mod"},
	st_reference{"web", "delete_by_msgid"},
	st_reference{"web", "ban_by_msgid"},
	st_reference{"web", "bname_topts_by_tid"},
	st_reference{"web", "refresh_bump_by_tid"},

	st_reference{"web", "set_mod_priv"},
	st_reference{"web", "fetch_and_clear_mod_msgs"},
}

func loadStatements() {
	if st_listx[0] != "" {
		panic("already loaded")
	}
	bm := make(map[string]sqlbucket.Bucket)
	for i := range st_names {
		sn := st_names[i]
		if bm[sn.Bucket] == nil {
			fn := "aux/psqlib/" + sn.Bucket + ".sql"
			stmts, err := sqlbucket.LoadFromFile(fn)
			if err != nil {
				st_loaderr = fmt.Errorf("err loading %s: %v", fn, err)
				return
			}
			bm[sn.Bucket] = stmts
		}
		sm := bm[sn.Bucket]
		sl := sm[sn.Name]
		if len(sl) != 1 {
			st_loaderr = fmt.Errorf(
				"wrong count %d for statement %s", len(sl), sn)
			return
		}
		st_listx[i] = sl[0] + "\n"
	}
}

var st_once sync.Once

func (sp *PSQLIB) prepareStatements() (err error) {
	if sp.st_prep[0] != nil {
		panic("already prepared")
	}
	for i := range st_listx {
		sp.st_prep[i], err = sp.db.DB.Prepare(st_listx[i])
		if err != nil {
			return fmt.Errorf("error preparing %d %q statement: %v",
				i, st_names[i].Name, err)
		}
	}
	return
}
