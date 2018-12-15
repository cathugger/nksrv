package mail

import (
	"sort"
	"strings"
)

// most of things there are copypasted from stdlib

// isTSpecial reports whether rune is in 'tspecials' as defined by RFC
// 1521 and RFC 2045.
func isTSpecial(r rune) bool {
	return strings.ContainsRune(`()<>@,;:\"/[]?=`, r)
}

// isTokenChar reports whether rune is in 'token' as defined by RFC
// 1521 and RFC 2045.
func isTokenChar(r rune) bool {
	// token := 1*<any (US-ASCII) CHAR except SPACE, CTLs,
	//             or tspecials>
	return r > 0x20 && r < 0x7f && !isTSpecial(r)
}

func isNotTokenChar(r rune) bool {
	return !isTokenChar(r)
}

// isToken reports whether s is a 'token' as defined by RFC 1521
// and RFC 2045.
func isToken(s string) bool {
	if s == "" {
		return false
	}
	return strings.IndexFunc(s, isNotTokenChar) < 0
}

const upperhex = "0123456789ABCDEF"

// FormatMediaTypeX serializes mediatype t and the parameters
// param as a media type conforming to RFC 2045 and RFC 2616.
// The type and parameter names are written in lower-case.
// When any of the arguments result in a standard violation then
// FormatMediaTypeX returns the empty string.
func FormatMediaTypeX(t string, param map[string]string) string {
	var b strings.Builder
	if slash := strings.Index(t, "/"); slash == -1 {
		if !isToken(t) {
			return ""
		}
		b.WriteString(strings.ToLower(t))
	} else {
		major, sub := t[:slash], t[slash+1:]
		if !isToken(major) || !isToken(sub) {
			return ""
		}
		b.WriteString(strings.ToLower(major))
		b.WriteByte('/')
		b.WriteString(strings.ToLower(sub))
	}

	attrs := make([]string, 0, len(param))
	for a := range param {
		attrs = append(attrs, a)
	}
	sort.Strings(attrs)

	for _, attribute := range attrs {
		value := param[attribute]
		b.WriteByte(';')
		b.WriteByte(' ')
		if !isToken(attribute) {
			return ""
		}
		b.WriteString(strings.ToLower(attribute))

		needsEscaping := false
		needsQuoting := value == "" // if value is empty, it needs quoting
		for idx := 0; idx < len(value); idx++ {
			ch := value[idx]
			// UTF-8, but also non-WSP CTLs, because they're
			// not allowed in RFC 5322 quoted-string
			if ch >= 0x7F || (ch < 0x20 && ch != '\t') {
				needsEscaping = true
				break
			}
			if !isTokenChar(rune(ch)) {
				needsQuoting = true
			}
		}

		if needsEscaping {
			b.WriteByte('*')
		}

		b.WriteByte('=')

		if needsEscaping {
			b.WriteString("utf-8''")

			offset := 0
			for index := 0; index < len(value); index++ {
				ch := value[index]
				// {RFC 2231 section 7}
				// attribute-char := <any (US-ASCII) CHAR except SPACE, CTLs, "*", "'", "%", or tspecials>
				if ch <= 0x20 || ch >= 0x7F ||
					ch == '*' || ch == '\'' || ch == '%' ||
					isTSpecial(rune(ch)) {

					b.WriteString(value[offset:index])
					offset = index + 1

					b.WriteByte('%')
					b.WriteByte(upperhex[ch>>4])
					b.WriteByte(upperhex[ch&0x0F])
				}
			}
			if offset < len(value) {
				b.WriteString(value[offset:])
			}
			continue
		}

		if !needsQuoting {
			b.WriteString(value)
			continue
		}

		b.WriteByte('"')
		offset := 0
		for index := 0; index < len(value); index++ {
			character := value[index]
			if character == '"' || character == '\\' {
				b.WriteString(value[offset:index])
				offset = index
				b.WriteByte('\\')
			}
		}
		b.WriteString(value[offset:])
		b.WriteByte('"')
	}
	return b.String()
}
