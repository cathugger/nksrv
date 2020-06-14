package pibase

import (
	"fmt"
	"strconv"

	"github.com/lib/pq"

	"nksrv/lib/sqlbucket"
)

const (
	St_nntp_article_exists_or_banned_by_msgid = iota
	St_nntp_article_valid_by_msgid

	St_nntp_article_num_by_msgid
	St_nntp_article_msgid_by_num

	St_nntp_article_get_gpid

	St_nntp_select
	St_nntp_select_and_list

	St_nntp_next
	St_nntp_last

	St_nntp_newnews_all
	St_nntp_newnews_one
	St_nntp_newnews_all_group

	St_nntp_newgroups

	St_nntp_listactive_all
	St_nntp_listactive_one

	St_nntp_over_msgid
	St_nntp_over_range
	St_nntp_over_curr

	St_nntp_hdr_msgid_msgid
	St_nntp_hdr_msgid_subject
	St_nntp_hdr_msgid_any
	St_nntp_hdr_range_msgid
	St_nntp_hdr_range_subject
	St_nntp_hdr_range_any
	St_nntp_hdr_curr_msgid
	St_nntp_hdr_curr_subject
	St_nntp_hdr_curr_any

	St_web_listboards
	St_web_thread_list_page
	St_web_overboard_page
	St_web_thread_catalog
	St_web_overboard_catalog
	St_web_thread

	St_web_prepost_newthread
	St_web_prepost_newpost

	St_mod_ref_write
	St_mod_ref_find_post
	St_mod_update_bpost_activ_refs

	St_mod_autoregister_mod
	St_mod_delete_by_msgid
	St_mod_ban_by_msgid
	St_mod_bname_topts_by_tid
	St_mod_refresh_bump_by_tid

	St_mod_set_mod_priv
	St_mod_set_mod_priv_group
	St_mod_unset_mod
	St_mod_fetch_and_clear_mod_msgs_start
	St_mod_fetch_and_clear_mod_msgs_continue

	St_mod_load_files

	St_mod_check_article_for_push
	St_mod_delete_ph_for_push
	St_mod_add_ph_after_push

	St_mod_joblist_modlist_changes_get
	St_mod_joblist_modlist_changes_set
	St_mod_joblist_modlist_changes_del

	St_mod_joblist_refs_deps_recalc_get
	St_mod_joblist_refs_deps_recalc_set
	St_mod_joblist_refs_deps_recalc_del

	St_mod_joblist_refs_recalc_get

	St_puller_get_last_newnews
	St_puller_set_last_newnews
	St_puller_get_last_newsgroups
	St_puller_set_last_newsgroups
	St_puller_get_group_id
	St_puller_set_group_id
	St_puller_unset_group_id
	St_puller_load_temp_groups

	stMax
)

var StListX [stMax]string
var StLoadErr error

type StReference struct {
	Bucket string
	Name   string
}

var stNames = [stMax]StReference{

	// NNTP stuff

	StReference{"nntp", "nntp_article_exists_or_banned_by_msgid"},
	StReference{"nntp", "nntp_article_valid_by_msgid"},

	StReference{"nntp", "nntp_article_num_by_msgid"},
	StReference{"nntp", "nntp_article_msgid_by_num"},

	StReference{"nntp", "nntp_article_get_gpid"},

	StReference{"nntp", "nntp_select"},
	StReference{"nntp", "nntp_select_and_list"},

	StReference{"nntp", "nntp_next"},
	StReference{"nntp", "nntp_last"},

	StReference{"nntp", "nntp_newnews_all"},
	StReference{"nntp", "nntp_newnews_one"},
	StReference{"nntp", "nntp_newnews_all_group"},

	StReference{"nntp", "nntp_newgroups"},

	StReference{"nntp", "nntp_listactive_all"},
	StReference{"nntp", "nntp_listactive_one"},

	StReference{"nntp", "nntp_over_msgid"},
	StReference{"nntp", "nntp_over_range"},
	StReference{"nntp", "nntp_over_curr"},

	StReference{"nntp", "nntp_hdr_msgid_msgid"},
	StReference{"nntp", "nntp_hdr_msgid_subject"},
	StReference{"nntp", "nntp_hdr_msgid_any"},
	StReference{"nntp", "nntp_hdr_range_msgid"},
	StReference{"nntp", "nntp_hdr_range_subject"},
	StReference{"nntp", "nntp_hdr_range_any"},
	StReference{"nntp", "nntp_hdr_curr_msgid"},
	StReference{"nntp", "nntp_hdr_curr_subject"},
	StReference{"nntp", "nntp_hdr_curr_any"},

	// web stuff

	StReference{"web", "web_listboards"},
	StReference{"web", "web_thread_list_page"},
	StReference{"web", "web_overboard_page"},
	StReference{"web", "web_thread_catalog"},
	StReference{"web", "web_overboard_catalog"},
	StReference{"web", "web_thread"},

	StReference{"web", "web_prepost_newthread"},
	StReference{"web", "web_prepost_newpost"},

	// database-modification

	StReference{"mod", "mod_ref_write"},
	StReference{"mod", "mod_ref_find_post"},
	StReference{"mod", "mod_update_bpost_activ_refs"},

	StReference{"mod", "mod_autoregister_mod"},
	StReference{"mod", "mod_delete_by_msgid"},
	StReference{"mod", "mod_ban_by_msgid"},
	StReference{"mod", "mod_bname_topts_by_tid"},
	StReference{"mod", "mod_refresh_bump_by_tid"},

	StReference{"mod", "mod_set_mod_priv"},
	StReference{"mod", "mod_set_mod_priv_group"},
	StReference{"mod", "mod_unset_mod"},
	StReference{"mod", "mod_fetch_and_clear_mod_msgs_start"},
	StReference{"mod", "mod_fetch_and_clear_mod_msgs_continue"},

	StReference{"mod", "mod_load_files"},

	StReference{"mod", "mod_check_article_for_push"},
	StReference{"mod", "mod_delete_ph_for_push"},
	StReference{"mod", "mod_add_ph_after_push"},

	// job list management

	StReference{"mod_joblist", "mod_joblist_modlist_changes_get"},
	StReference{"mod_joblist", "mod_joblist_modlist_changes_set"},
	StReference{"mod_joblist", "mod_joblist_modlist_changes_del"},

	StReference{"mod_joblist", "mod_joblist_refs_deps_recalc_get"},
	StReference{"mod_joblist", "mod_joblist_refs_deps_recalc_set"},
	StReference{"mod_joblist", "mod_joblist_refs_deps_recalc_del"},

	StReference{"mod_joblist", "mod_joblist_refs_recalc_get"},

	// puller-related

	StReference{"puller", "puller_get_last_newnews"},
	StReference{"puller", "puller_set_last_newnews"},
	StReference{"puller", "puller_get_last_newsgroups"},
	StReference{"puller", "puller_set_last_newsgroups"},
	StReference{"puller", "puller_get_group_id"},
	StReference{"puller", "puller_set_group_id"},
	StReference{"puller", "puller_unset_group_id"},
	StReference{"puller", "puller_load_temp_groups"},
}

func LoadStatements() {
	if StListX[0] != "" {
		panic("already loaded")
	}
	bm := make(map[string]sqlbucket.Bucket)
	for i := range stNames {
		sn := stNames[i]
		if bm[sn.Bucket] == nil {
			fn := "etc/psqlib/" + sn.Bucket + ".sql"
			stmts, err := sqlbucket.LoadFromFile(fn)
			if err != nil {
				StLoadErr = fmt.Errorf("err loading %s: %v", fn, err)
				return
			}
			bm[sn.Bucket] = stmts
		}
		sm := bm[sn.Bucket]
		sl := sm[sn.Name]
		if len(sl) != 1 {
			StLoadErr = fmt.Errorf(
				"wrong count %d for statement %s", len(sl), sn)
			return
		}
		StListX[i] = sl[0] + "\n"
	}
}

func (sp *PSQLIB) prepareStatements() (err error) {
	if sp.StPrep[0] != nil {
		panic("already prepared")
	}
	for i := range StListX {

		s := StListX[i]
		sp.StPrep[i], err = sp.DB.DB.Prepare(s)
		if err != nil {

			if pe, _ := err.(*pq.Error); pe != nil {

				pos, _ := strconv.Atoi(pe.Position)
				ss, se := pos, pos
				for ss > 0 && s[ss-1] != '\n' {
					ss--
				}
				for se < len(s) && s[se] != '\n' {
					se++
				}

				return fmt.Errorf(
					"err preparing %d %q stmt: pos[%s] msg[%s] detail[%s] line[%s]\nstmt:\n%s",
					i, stNames[i].Name, pe.Position, pe.Message, pe.Detail, s[ss:se], s)
			}

			return fmt.Errorf("error preparing %d %q stmt: %v",
				i, stNames[i].Name, err)
		}
	}
	return
}

func (sp *PSQLIB) closeStatements() (err error) {
	for i := range StListX {
		if sp.StPrep[i] != nil {
			ex := sp.StPrep[i].Close()
			if err == nil {
				err = ex
			}
			sp.StPrep[i] = nil
		}
	}
	return
}
