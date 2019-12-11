package psqlib

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	. "nksrv/lib/logx"
	"nksrv/lib/mail/form"
	"nksrv/lib/mailib"
	ib0 "nksrv/lib/webib0"
)

func readableText(s string) bool {
	for _, c := range s {
		if (c < 32 && c != '\n' && c != '\r' && c != '\t') || c == 127 {
			return false
		}
	}
	return true
}

var lineReplacer = strings.NewReplacer(
	"\r", "",
	"\n", " ",
	"\t", " ",
	"\000", "")

func optimiseFormLine(line string) (s string) {
	s = lineReplacer.Replace(line)
	s = norm.NFC.String(s)
	return
}

func checkTextLimits(
	slimits *submissionLimits, reply bool,
	mInfo mailib.MessageInfo) error {

	if len(mInfo.Title) > int(slimits.MaxTitleLength) {
		return errTooLongTitle
	}
	if len(mInfo.Author) > int(slimits.MaxNameLength) {
		return errTooLongName
	}
	if len(mInfo.Message) > int(slimits.MaxMessageLength) {
		return errTooLongMessage(slimits.MaxMessageLength)
	}

	return nil
}

type webInputFields struct {
	title   string
	name    string
	message string
	options string
}

func (sp *PSQLIB) processTextFields(
	f form.Form) (xf webInputFields, err error) {

	// field names
	fntitle := ib0.IBWebFormTextTitle
	fnname := ib0.IBWebFormTextName
	fnmessage := ib0.IBWebFormTextMessage
	fnoptions := ib0.IBWebFormTextOptions

	// check field counts
	if len(f.Values[fntitle]) > 1 ||
		len(f.Values[fnname]) != 1 ||
		len(f.Values[fnmessage]) != 1 ||
		len(f.Values[fnoptions]) > 1 {

		err = errInvalidSubmission
		return
	}

	// assign
	if len(f.Values[fntitle]) != 0 {
		xf.title = f.Values[fntitle][0]
	}
	xf.name = f.Values[fnname][0]
	xf.message = f.Values[fnmessage][0]
	if len(f.Values[fnoptions]) != 0 {
		xf.options = f.Values[fnoptions][0]
	}

	// print
	sp.log.LogPrintf(
		DEBUG,
		"post: xftitle %q xfmessage %q xfoptions %q",
		xf.title, xf.message, xf.options)

	// validate encoding
	if !utf8.ValidString(xf.title) ||
		!utf8.ValidString(xf.name) ||
		!utf8.ValidString(xf.message) ||
		!utf8.ValidString(xf.options) {

		err = errBadSubmissionEncoding
		return
	}

	// validate if OK chars are used inside
	if !readableText(xf.title) ||
		!readableText(xf.name) ||
		!readableText(xf.message) ||
		!readableText(xf.options) {

		err = errBadSubmissionChars
		return
	}

	// all good
	return
}
