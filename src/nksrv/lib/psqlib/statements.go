package psqlib

import (
	"fmt"
	"sync"

	"nksrv/lib/sqlbucket"
)

const (
	st_nntp_article_exists_or_banned_by_msgid = iota
	st_nntp_article_valid_and_banned_by_msgid

	st_nntp_article_num_by_msgid
	st_nntp_article_msgid_by_num

	st_nntp_article_get_gpid

	st_nntp_select
	st_nntp_select_and_list

	st_nntp_next
	st_nntp_last

	st_nntp_newnews_all
	st_nntp_newnews_one
	st_nntp_newnews_all_group

	st_nntp_newgroups

	st_nntp_listactive_all
	st_nntp_listactive_one

	st_nntp_over_msgid
	st_nntp_over_range
	st_nntp_over_curr

	st_nntp_hdr_msgid_msgid
	st_nntp_hdr_msgid_subject
	st_nntp_hdr_msgid_any
	st_nntp_hdr_range_msgid
	st_nntp_hdr_range_subject
	st_nntp_hdr_range_any
	st_nntp_hdr_curr_msgid
	st_nntp_hdr_curr_subject
	st_nntp_hdr_curr_any

	st_web_listboards
	st_web_thread_list_page
	st_web_overboard_page
	st_web_thread_catalog
	st_web_overboard_catalog
	st_web_thread

	st_web_prepost_newthread
	st_web_prepost_newpost

	st_mod_ref_write
	st_mod_ref_find_post
	st_mod_update_bpost_activ_refs

	st_mod_autoregister_mod
	st_mod_delete_by_msgid
	st_mod_delete_by_gpid
	st_mod_ban_by_msgid
	st_mod_bname_topts_by_tid
	st_mod_refresh_bump_by_tid

	st_mod_set_mod_priv
	st_mod_unset_mod
	st_mod_fetch_and_clear_mod_msgs

	st_mod_load_files

	st_puller_get_last_newnews
	st_puller_set_last_newnews
	st_puller_get_last_newsgroups
	st_puller_set_last_newsgroups
	st_puller_get_group_id
	st_puller_set_group_id
	st_puller_unset_group_id
	st_puller_load_temp_groups

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
	st_reference{"web", "web_overboard_catalog"},
	st_reference{"web", "web_thread"},

	st_reference{"web", "web_prepost_newthread"},
	st_reference{"web", "web_prepost_newpost"},

	st_reference{"mod", "mod_ref_write"},
	st_reference{"mod", "mod_ref_find_post"},
	st_reference{"mod", "mod_update_bpost_activ_refs"},

	st_reference{"mod", "mod_autoregister_mod"},
	st_reference{"mod", "mod_delete_by_msgid"},
	st_reference{"mod", "mod_delete_by_gpid"},
	st_reference{"mod", "mod_ban_by_msgid"},
	st_reference{"mod", "mod_bname_topts_by_tid"},
	st_reference{"mod", "mod_refresh_bump_by_tid"},

	st_reference{"mod", "mod_set_mod_priv"},
	st_reference{"mod", "mod_unset_mod"},
	st_reference{"mod", "mod_fetch_and_clear_mod_msgs"},

	st_reference{"mod", "mod_load_files"},

	st_reference{"puller", "puller_get_last_newnews"},
	st_reference{"puller", "puller_set_last_newnews"},
	st_reference{"puller", "puller_get_last_newsgroups"},
	st_reference{"puller", "puller_set_last_newsgroups"},
	st_reference{"puller", "puller_get_group_id"},
	st_reference{"puller", "puller_set_group_id"},
	st_reference{"puller", "puller_unset_group_id"},
	st_reference{"puller", "puller_load_temp_groups"},
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

func (sp *PSQLIB) closeStatements() (err error) {
	for i := range st_listx {
		if sp.st_prep[i] != nil {
			ex := sp.st_prep[i].Close()
			if err == nil {
				err = ex
			}
			sp.st_prep[i] = nil
		}
	}
	return
}
