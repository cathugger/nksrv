package nntppostlockmgr

/*
 * we want to prevent multiple article insertions
 * but simplest implementation of this would allow DoS by starting to send article but never finishing it
 * therefore we must have some sort of timeout for this
 * we also need to keep track of posting failures, otherwise timeout could be easily defeated
 * probably best way to handle this would be keeping entry about posting failure and poster, allow that poster to try resending article, but don't lock off other posters
 * but then posters could flood lock manager by sending tons of bad articles; we should keep track of posters who do that and after certain threshold discard all their expired locks and disallow them taking any more locks
 * this stuff should be tracker per-connection or per-poster basis? not decided (what is poster's identity?)
 * POST cmd is a bit edge case as it doesn't ask for Message-ID upfront, but there is no real harm of submitting article
 * we SHOULD handle conflicting inserts gracefuly regardless of post locks
 */

const (
	stNone
	stCanceled
	stPosting
)
