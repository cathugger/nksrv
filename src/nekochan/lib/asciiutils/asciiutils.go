package asciiutils

// EqualFoldString is basically strcasecmp.
func EqualFoldString(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ac, bc := a[i], b[i]
		if ac == bc {
			continue
		}
		if ac > bc {
			// ensure ac < bc
			ac, bc = bc, ac
		}
		if ac >= 'A' && ac <= 'Z' && ac+'a'-'A' == bc {
			continue
		}
		return false
	}
	return true
}

// StartsWithFoldString checks if b starts with s in case-insensitive way.
func StartsWithFoldString(b, s string) bool {
	if len(b) < len(s) {
		// b cant start with s if b is shorter
		return false
	}
	// use s length
	for i := 0; i < len(s); i++ {
		ac, bc := b[i], s[i]
		if ac == bc {
			continue
		}
		if ac > bc {
			// ensure ac < bc
			ac, bc = bc, ac
		}
		if ac >= 'A' && ac <= 'Z' && ac+'a'-'A' == bc {
			continue
		}
		return false
	}
	return true
}

// EndsWithFoldString checks if b ends with s in case-insensitive way.
func EndsWithFoldString(b, s string) bool {
	if len(b) < len(s) {
		// b cant end with s if b is shorter
		return false
	}
	// use s length
	for i := 1; i <= len(s); i++ {
		ac, bc := b[len(b)-i], s[len(s)-i]
		if ac == bc {
			continue
		}
		if ac > bc {
			// ensure ac < bc
			ac, bc = bc, ac
		}
		if ac >= 'A' && ac <= 'Z' && ac+'a'-'A' == bc {
			continue
		}
		return false
	}
	return true
}

func UntilString(s string, c byte) string {
	i := 0
	for ; i < len(s) && s[i] != c; i++ {
	}
	return s[:i]
}

func IterateFields(s string, f func(string)) (n int) {
	i := 0
	for {
		// skip space
		for ; i < len(s) && (s[i] == ' ' || s[i] == '\t'); i++ {
		}
		// reached the end?
		if i >= len(s) {
			return
		}
		is := i
		// skip to space or end
		for ; i < len(s) && s[i] != ' ' && s[i] != '\t'; i++ {
		}

		f(s[is:i])
		n++
	}
}

func TrimWSString(b string) string {
	x, y := 0, len(b)
	for x != len(b) && (b[x] == ' ' || b[x] == '\t') {
		x++
	}
	for y != x && (b[y-1] == ' ' || b[y-1] == '\t') {
		y--
	}
	return b[x:y]
}

func TrimWSBytes(b []byte) []byte {
	x, y := 0, len(b)
	for x != len(b) && (b[x] == ' ' || b[x] == '\t') {
		x++
	}
	for y != x && (b[y-1] == ' ' || b[y-1] == '\t') {
		y--
	}
	return b[x:y]
}

// NOTE ASCII space (32) is neither printable chatacter nor control character
func IsPrintableASCIISlice(s []byte, e byte) bool {
	for _, c := range s {
		if c <= 32 || c >= 127 || c == e {
			return false
		}
	}
	return true
}

func Is7BitBytes(s []byte) bool {
	for _, c := range s {
		if c >= 0x80 {
			return false
		}
	}
	return true
}

func Is7BitString(s string) bool {
	for _, c := range []byte(s) {
		if c >= 0x80 {
			return false
		}
	}
	return true
}
