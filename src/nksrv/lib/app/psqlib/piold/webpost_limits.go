package psqlib

import (
	"nksrv/lib/app/mailib"
	"nksrv/lib/mail/form"
)

func checkSubmissionLimits(
	slimits *submissionLimits, reply bool,
	f form.Form, mInfo mailib.MessageInfo) (
	err error, c int) {

	err, c = checkFileLimits(slimits, reply, f)
	if err != nil {
		return
	}

	err = checkTextLimits(slimits, reply, mInfo)
	return
}

func (sp *PSQLIB) applyInstanceSubmissionLimits(
	slimits *submissionLimits, reply bool, board string) {

	// TODO

	// hardcoded instance limits, TODO make configurable

	if slimits.MaxTitleLength == 0 || slimits.MaxTitleLength > maxSubjectSize {
		slimits.MaxTitleLength = maxSubjectSize
	}

	if slimits.MaxNameLength == 0 || slimits.MaxNameLength > maxNameSize {
		slimits.MaxNameLength = maxNameSize
	}

	const maxMessageLength = mailib.DefaultMaxTextLen
	if slimits.MaxMessageLength == 0 ||
		slimits.MaxMessageLength > maxMessageLength {

		slimits.MaxMessageLength = maxMessageLength
	}
}

func (sp *PSQLIB) applyInstanceThreadOptions(
	threadOpts *threadOptions, board string) {

	// TODO
}
