package pibase

import (
	"fmt"
	"strconv"

	"github.com/lib/pq"

	"nksrv/lib/sqlbucket"
)

const (

	// NNTP

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

	// web

	St_web_listboards
	St_web_thread_list_page
	St_web_overboard_page
	St_web_thread_catalog
	St_web_overboard_catalog
	St_web_thread

	St_web_prepost_newthread
	St_web_prepost_newpost

	// post

	St_post_newthread_sb_nf
	St_post_newthread_mb_nf
	St_post_newthread_sb_sf
	St_post_newthread_mb_sf
	St_post_newthread_sb_mf
	St_post_newthread_mb_mf

	St_post_newreply_sb_nf
	St_post_newreply_mb_nf
	St_post_newreply_sb_sf
	St_post_newreply_mb_sf
	St_post_newreply_sb_mf
	St_post_newreply_mb_mf

	// various modification

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

	// joblist

	St_mod_joblist_modlist_changes_get
	St_mod_joblist_modlist_changes_set
	St_mod_joblist_modlist_changes_del

	St_mod_joblist_refs_deps_recalc_get
	St_mod_joblist_refs_deps_recalc_set
	St_mod_joblist_refs_deps_recalc_del

	St_mod_joblist_refs_recalc_get

	// puller specific

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

	{"nntp", "nntp_article_exists_or_banned_by_msgid"},
	{"nntp", "nntp_article_valid_by_msgid"},

	{"nntp", "nntp_article_num_by_msgid"},
	{"nntp", "nntp_article_msgid_by_num"},

	{"nntp", "nntp_article_get_gpid"},

	{"nntp", "nntp_select"},
	{"nntp", "nntp_select_and_list"},

	{"nntp", "nntp_next"},
	{"nntp", "nntp_last"},

	{"nntp", "nntp_newnews_all"},
	{"nntp", "nntp_newnews_one"},
	{"nntp", "nntp_newnews_all_group"},

	{"nntp", "nntp_newgroups"},

	{"nntp", "nntp_listactive_all"},
	{"nntp", "nntp_listactive_one"},

	{"nntp", "nntp_over_msgid"},
	{"nntp", "nntp_over_range"},
	{"nntp", "nntp_over_curr"},

	{"nntp", "nntp_hdr_msgid_msgid"},
	{"nntp", "nntp_hdr_msgid_subject"},
	{"nntp", "nntp_hdr_msgid_any"},
	{"nntp", "nntp_hdr_range_msgid"},
	{"nntp", "nntp_hdr_range_subject"},
	{"nntp", "nntp_hdr_range_any"},
	{"nntp", "nntp_hdr_curr_msgid"},
	{"nntp", "nntp_hdr_curr_subject"},
	{"nntp", "nntp_hdr_curr_any"},

	// web stuff

	{"web", "web_listboards"},
	{"web", "web_thread_list_page"},
	{"web", "web_overboard_page"},
	{"web", "web_thread_catalog"},
	{"web", "web_overboard_catalog"},
	{"web", "web_thread"},

	{"web", "web_prepost_newthread"},
	{"web", "web_prepost_newpost"},

	// post stuff

	{"post", "post_newthread_sb_nf"},
	{"post", "post_newthread_mb_nf"},
	{"post", "post_newthread_sb_sf"},
	{"post", "post_newthread_mb_sf"},
	{"post", "post_newthread_sb_mf"},
	{"post", "post_newthread_mb_mf"},

	{"post", "post_newreply_sb_nf"},
	{"post", "post_newreply_mb_nf"},
	{"post", "post_newreply_sb_sf"},
	{"post", "post_newreply_mb_sf"},
	{"post", "post_newreply_sb_mf"},
	{"post", "post_newreply_mb_mf"},

	// database-modification

	{"mod", "mod_ref_write"},
	{"mod", "mod_ref_find_post"},
	{"mod", "mod_update_bpost_activ_refs"},

	{"mod", "mod_autoregister_mod"},
	{"mod", "mod_delete_by_msgid"},
	{"mod", "mod_ban_by_msgid"},
	{"mod", "mod_bname_topts_by_tid"},
	{"mod", "mod_refresh_bump_by_tid"},

	{"mod", "mod_set_mod_priv"},
	{"mod", "mod_set_mod_priv_group"},
	{"mod", "mod_unset_mod"},
	{"mod", "mod_fetch_and_clear_mod_msgs_start"},
	{"mod", "mod_fetch_and_clear_mod_msgs_continue"},

	{"mod", "mod_load_files"},

	{"mod", "mod_check_article_for_push"},
	{"mod", "mod_delete_ph_for_push"},
	{"mod", "mod_add_ph_after_push"},

	// job list management

	{"mod_joblist", "mod_joblist_modlist_changes_get"},
	{"mod_joblist", "mod_joblist_modlist_changes_set"},
	{"mod_joblist", "mod_joblist_modlist_changes_del"},

	{"mod_joblist", "mod_joblist_refs_deps_recalc_get"},
	{"mod_joblist", "mod_joblist_refs_deps_recalc_set"},
	{"mod_joblist", "mod_joblist_refs_deps_recalc_del"},

	{"mod_joblist", "mod_joblist_refs_recalc_get"},

	// puller-related

	{"puller", "puller_get_last_newnews"},
	{"puller", "puller_set_last_newnews"},
	{"puller", "puller_get_last_newsgroups"},
	{"puller", "puller_set_last_newsgroups"},
	{"puller", "puller_get_group_id"},
	{"puller", "puller_set_group_id"},
	{"puller", "puller_unset_group_id"},
	{"puller", "puller_load_temp_groups"},
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
