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
	return fmt.Sprintf(
		"%02d %s %04d %02d:%02d:%02d +0000", D, M.String()[:3], Y, h, m, s)
}
