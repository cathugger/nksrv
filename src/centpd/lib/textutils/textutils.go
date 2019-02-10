package textutils

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// stuff we don't wanna
var replacer = strings.NewReplacer(
	"\r", "",
	"\000", string(unicode.ReplacementChar))

func isNilOrSpace(r rune) bool {
	return unicode.IsSpace(r) || r == 0
}

func NormalizeTextMessage(msg string) (s string) {
	// normalise using form C
	s = norm.NFC.String(msg)
	// trim line endings, and empty lines at the end
	lines := strings.Split(s, "\n")
	for i, v := range lines {
		lines[i] = replacer.Replace(strings.TrimRightFunc(v, isNilOrSpace))
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			break
		}
		lines = lines[:i]
	}
	s = strings.Join(lines, "\n")
	return
}

// TruncateText truncates valid utf8 string
// to specified length or less without breaking it.
func TruncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for ; n != 0; n-- {
		// cut off only in places before next full rune
		if utf8.RuneStart(s[n]) {
			return s[:n]
		}
	}
	return ""
}
