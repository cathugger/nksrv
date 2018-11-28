package mail

import (
	"fmt"
	nmail "net/mail"
	"time"
)

func ParseDate(date string) (time.Time, error) {
	// im lazy
	return nmail.ParseDate(date)
}

func FormatDate(t time.Time) string {
	t = t.UTC()
	Y, M, D := t.Date()
	h, m, s := t.Clock()
	W := t.Weekday()
	// non-recent nntpchan (fixed in 4d4aea61fedc) is very inflexible with this
	// TODO axe out weekday when the time is right
	return fmt.Sprintf(
		"%s, %02d %s %04d %02d:%02d:%02d +0000",
		W.String()[:3], D, M.String()[:3], Y, h, m, s)
}
