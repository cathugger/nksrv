package basesql

//go:generate stringer -output statements_string.go -trimprefix SI_ -type=StatementIndexEntry

type StatementIndexEntry int

const (

	// nntp

	SI_nntp_article_exists_or_banned_by_msgid StatementIndexEntry = iota
	SI_nntp_article_valid_by_msgid

	SI_nntp_article_num_by_msgid
	SI_nntp_article_msgid_by_num

	SI_nntp_article_get_gpid

	SI_nntp_select
	SI_nntp_select_and_list

	SI_nntp_next
	SI_nntp_last

	SI_nntp_newnews_all
	SI_nntp_newnews_one
	SI_nntp_newnews_all_group

	SI_nntp_newgroups

	SI_nntp_listactive_all
	SI_nntp_listactive_one

	SI_nntp_over_msgid
	SI_nntp_over_range
	SI_nntp_over_curr

	SI_nntp_hdr_msgid_msgid
	SI_nntp_hdr_msgid_subject
	SI_nntp_hdr_msgid_any
	SI_nntp_hdr_range_msgid
	SI_nntp_hdr_range_subject
	SI_nntp_hdr_range_any
	SI_nntp_hdr_curr_msgid
	SI_nntp_hdr_curr_subject
	SI_nntp_hdr_curr_any

	// web

	SI_web_listboards
	SI_web_thread_list_page
	SI_web_overboard_page
	SI_web_thread_catalog
	SI_web_overboard_catalog
	SI_web_thread

	SI_web_prepost_newthread
	SI_web_prepost_newpost

	// post

	SI_post_newthread_sb_nf
	SI_post_newthread_mb_nf
	SI_post_newthread_sb_sf
	SI_post_newthread_mb_sf
	SI_post_newthread_sb_mf
	SI_post_newthread_mb_mf

	SI_post_newreply_sb_nf
	SI_post_newreply_mb_nf
	SI_post_newreply_sb_sf
	SI_post_newreply_mb_sf
	SI_post_newreply_sb_mf
	SI_post_newreply_mb_mf

	// various modification

	SI_mod_ref_write
	SI_mod_ref_find_post
	SI_mod_update_bpost_activ_refs

	SI_mod_autoregister_mod
	SI_mod_delete_by_msgid
	SI_mod_ban_by_msgid
	SI_mod_bname_topts_by_tid
	SI_mod_refresh_bump_by_tid

	SI_mod_set_mod_priv
	SI_mod_set_mod_priv_group
	SI_mod_unset_mod
	SI_mod_fetch_and_clear_mod_msgs_start
	SI_mod_fetch_and_clear_mod_msgs_continue

	SI_mod_load_files

	SI_mod_check_article_for_push
	SI_mod_delete_ph_for_push
	SI_mod_add_ph_after_push

	// joblist

	SI_mod_joblist_modlist_changes_get
	SI_mod_joblist_modlist_changes_set
	SI_mod_joblist_modlist_changes_del

	SI_mod_joblist_refs_deps_recalc_get
	SI_mod_joblist_refs_deps_recalc_set
	SI_mod_joblist_refs_deps_recalc_del

	SI_mod_joblist_refs_recalc_get

	// puller specific

	SI_puller_get_last_newnews
	SI_puller_set_last_newnews
	SI_puller_get_last_newsgroups
	SI_puller_set_last_newsgroups
	SI_puller_get_group_id
	SI_puller_set_group_id
	SI_puller_unset_group_id
	SI_puller_load_temp_groups

	SISize int = iota
)
