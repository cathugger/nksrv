package mail

import (
	"errors"
	"fmt"
	nmail "net/mail"
	"time"
)

func ParseDateX(date string) (t time.Time, err error) {
	// im lazy
	t, err = nmail.ParseDate(date)
	if err == nil {
		return
	}
	// try some more
	fallbacks := []string{
		"02 Jan 2006 15:04:05",
	}
	for _, l := range fallbacks {
		t, err = time.Parse(l, date)
		if err == nil {
			return
		}
	}
	return time.Time{}, errors.New("unsupported date format")
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
