package tmplrenderer

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

var funcs = map[string]interface{}{
	"urlpath":    urlPath,
	"truncatefn": truncatefn,
	"filesize":   filesize,
	"date":       date,
	"fmtmsg":     fmtmsg,
}

func urlPath(p string) string {
	return (&url.URL{Path: p}).EscapedPath()
}

func truncatefn(s string, l int) string {
	if utf8.RuneCountInString(s) <= l {
		// fast path, no truncation needed
		return s
	}
	i := strings.LastIndexByte(s, '.')
	// assume extension isnt special snowflake utf8
	// if there is no dot or len("(...).ext") would exceed our limits
	if i < 0 || 5+(len(s)-i) > l {
		// use "filename..." form instead which doesnt give special treatment to extension
		canuse := l - 3
		x, j := 0, 0
		for j = range s {
			if x >= canuse {
				break
			}
			x++
		}
		return s[:j] + "..."
	}
	// use "fn(...).ext" form
	canuse := l - 5 - (len(s) - i)
	x, j := 0, 0
	for j = range s {
		if x >= canuse {
			break
		}
		x++
	}
	return s[:j] + "(...)" + s[i:]
}

func filesize(s int64) string {
	if s < 1<<10 {
		return fmt.Sprintf("%d B", s)
	}
	fs := float64(s)
	if s < 1<<20 {
		return fmt.Sprintf("%.3f KiB", fs/(1<<10))
	}
	if s < 1<<30 {
		return fmt.Sprintf("%.3f MiB", fs/(1<<20))
	}
	if s < 1<<40 {
		return fmt.Sprintf("%.3f GiB", fs/(1<<30))
	}
	return fmt.Sprintf("%.6f TiB", fs/(1<<40))
}

func date(u int64) string {
	t := time.Unix(u, 0)
	Y, M, D := t.Date()
	h, m, s := t.Hour(), t.Minute(), t.Second()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", Y, M, D, h, m, s)
}
