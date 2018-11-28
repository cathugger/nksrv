package textutils

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func NormalizeTextMessage(msg string) (s string) {
	// normalise using form C
	s = norm.NFC.String(msg)
	// trim line endings, and empty lines at the end
	lines := strings.Split(s, "\n")
	for i, v := range lines {
		lines[i] = strings.TrimRightFunc(v, unicode.IsSpace)
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			break
		}
		lines = lines[:i]
	}
	s = strings.Join(lines, "\n")
	// ensure we don't have any CR left
	s = strings.Replace(s, "\r", "", -1)
	return
}
