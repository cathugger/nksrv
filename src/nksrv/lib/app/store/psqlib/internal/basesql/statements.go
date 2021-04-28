package basesql

//go:generate stringer -trimprefix St_ -type=statementIndexType

type statementIndexType int

const (

	// NNTP

	St_nntp_article_exists_or_banned_by_msgid statementIndexType = iota
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

	StMax int = iota
)
