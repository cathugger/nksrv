-- :set version 0.1.0

CREATE SCHEMA news;

CREATE TABLE news.gposts (
	g_p_id  BIGINT  GENERATED ALWAYS AS IDENTITY, -- global internal post ID
	msgid   TEXT    COLLATE "C"  NOT NULL,        -- Message-ID

    date_recv  TIMESTAMP  WITH TIME ZONE,
	date_sent  TIMESTAMP  WITH TIME ZONE,

	headers  JSONB, -- headers of msg root, map of lists of strings, needed for NNTP HDR

    article_file  TEXT  COLLATE "C"  NOT NULL, -- file name of article (usually hash of Message-ID)


	PRIMARY KEY (g_p_id),
	UNIQUE      (msgid)
);


CREATE TABLE news.boards (
	b_id      INTEGER  GENERATED ALWAYS AS IDENTITY, -- internal board ID
	newsgroup TEXT     COLLATE "C",                  -- newsgroup name
	last_id   BIGINT   DEFAULT 0    NOT NULL,        -- used for post/thread IDs

	p_count BIGINT  DEFAULT 0  NOT NULL, -- article count

	badded TIMESTAMP  WITH TIME ZONE  NOT NULL, -- date added to our node
	bdesc  TEXT                       NOT NULL, -- short description

	threads_per_page INTEGER, -- <=0 - infinite, this results in only single page
	max_active_pages INTEGER, -- <=0 - all existing pages are active
	max_pages        INTEGER, -- <=0 - unlimited, archive mode

	post_limits      JSONB, -- allowed properties of post, sorta common for both OPs and replies
	newthread_limits JSONB, -- same as post_limits but for new threads. inherits from post_limits
	reply_limits     JSONB, -- same as post_limits but for replies. inherits from post_limits
	thread_opts      JSONB, -- options common for all threads. stuff like bump/file limits
	attrib           JSONB, -- board attributes


	PRIMARY KEY (b_id),
	UNIQUE      (b_name)
)
-- :next
CREATE INDEX
	ON ib0.boards (badded,b_id) -- NEWGROUPS
-- :next
-- for UI-visible board list
CREATE INDEX
	ON ib0.boards (b_name COLLATE "und-x-icu")
	WHERE b_name IS NOT NULL
-- for netnews-visible grouplist
CREATE INDEX
	ON ib0.boards (newsgroup COLLATE "und-x-icu")
	WHERE newsgroup IS NOT NULL
