package tmplrenderer

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"

	"nksrv/lib/date"
	"nksrv/lib/srndtrip"
	tu "nksrv/lib/textutils"
	ib0 "nksrv/lib/webib0"
)

func f_list(args ...interface{}) []interface{} {
	return args
}

func f_dict(args ...interface{}) (m map[interface{}]interface{}, _ error) {
	if len(args)%2 != 0 {
		return nil, errors.New("odd number of arguments to map")
	}
	m = make(map[interface{}]interface{})
	for i := 0; i+1 < len(args); i += 2 {
		m[args[i]] = args[i+1]
	}
	return m, nil
}

var funcs = map[string]interface{}{
	// basics which should be there by default but aren't
	"list": f_list,
	"dict": f_dict,
	"map":  f_dict,
	"emptylist": func(v interface{}) ([]struct{}, error) {
		rv := reflect.ValueOf(v)
		var n int
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n = int(rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n = int(rv.Uint())
		default:
			return nil, errors.New("emptylist: passed value is not int")
		}
		return make([]struct{}, n), nil
	},
	"add_i": func(a, b int) int {
		return a + b
	},
	"add_u32": func(a, b uint32) uint32 {
		return a + b
	},
	// hacks
	"threadptr": func(x *ib0.IBCommonThread) *ib0.IBCommonThread {
		return x
	},
	"postptr": func(x *ib0.IBPostInfo) *ib0.IBPostInfo {
		return x
	},
	// stuff
	"urlpath":    urlPath,
	"escboard":   escBoard,
	"truncatefn": truncatefn,
	"filesize":   filesize,
	"fileinfo":   fileinfo,
	"filedata":   filedata,
	// normal display style, kinda inspired by RFC 3339
	"date": func(u int64) string {
		t := date.UnixTime(u)
		Y, M, D := t.Date()
		W := t.Weekday()
		h, m, s := t.Clock()
		return fmt.Sprintf("%04d-%02d-%02d (%s) %02d:%02d:%02d",
			Y, M, D, W.String()[:3], h, m, s)
	},
	// alternate display style, kinda format of RFC 2822 Date header
	"dateAlt": func(u int64) string {
		t := date.UnixTime(u)
		Y, M, D := t.Date()
		W := t.Weekday()
		h, m, s := t.Clock()
		return fmt.Sprintf("%s, %d %s %04d %02d:%02d:%02d",
			W, D, M, Y, h, m, s)
	},
	// ISO 8601
	"dateISO": func(u int64) string {
		t := date.UnixTimeUTC(u)
		Y, M, D := t.Date()
		h, m, s := t.Clock()
		return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ",
			Y, M, D, h, m, s)
	},
	"anonname": func(s string) string {
		// TODO configurable? per-board?
		if s == "" {
			return "Anonymous"
		}
		return s
	},
	"truncate": tu.TruncateText,
	"shortid": func(s string) string {
		// TODO configurable?
		const maxidlen = 20
		if len(s) > maxidlen {
			// we expect input to be US-ASCII
			return s[:maxidlen]
		} else {
			return s
		}
	},
	"unitrip":   srndtrip.MakeUnicodeTrip,
	"fmtmsg":    fmtmsg,
	"fmtmsgcat": fmtmsgcat,
}

func urlPath(p string) string {
	return (&url.URL{Path: p}).EscapedPath()
}

func escBoard(b string) string {
	b = urlPath(b)
	if len(b) != 0 && b[0] == '_' {
		b = "_" + b
	}
	return b
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

func fileinfo(fi *ib0.IBFileInfo) string {
	// width x height
	wf, wok := fi.Options["width"].(float64)
	hf, hok := fi.Options["height"].(float64)
	if wok && hok {
		return fmt.Sprintf("%dx%d, %s", int(wf), int(hf), filesize(fi.Size))
	}

	return filesize(fi.Size)
}

func filedata(fi *ib0.IBFileInfo) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, `data-type="%s"`, fi.Type)
	// XXX maybe we should just pass-thru these?
	if fi.Type == "image" {
		wf, wok := fi.Options["width"].(float64)
		hf, hok := fi.Options["height"].(float64)
		if wok && hok {
			fmt.Fprintf(b, ` data-width="%d"`, int(wf))
			fmt.Fprintf(b, ` data-height="%d"`, int(hf))
		}
	}
	return b.String()
}
