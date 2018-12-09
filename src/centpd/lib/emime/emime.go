package emime

import (
	"bufio"
	mm "mime"
	"os"
	"strings"
	"sync"
)

var (
	mimeLock       sync.RWMutex
	mimeTypes      map[string][]string // extension -> types
	mimeExtensions map[string][]string // type -> extensions
	mimePrefExt    map[string]string   // prefered extensions
)

// ext MUST be lowercase
func setExtensionType(
	ext, mimeType string) (pureExt string, pref bool, err error) {

	justType, param, err := mm.ParseMediaType(mimeType)
	if err != nil {
		return
	}
	// treat text files as UTF-8 by default
	if strings.HasPrefix(justType, "text/") && param["charset"] == "" {
		param["charset"] = "utf-8"
	}
	// ensure proper formatting
	mimeType = mm.FormatMediaType(justType, param)
	if len(ext) != 0 && (ext[0] == '.' || ext[0] == '!' || ext[0] == '=') {
		pureExt = ext[1:]
		if ext[0] == '=' {
			pref = true
		}
	} else {
		pureExt = ext
	}

	mtypes := mimeTypes[pureExt]
	for _, v := range mtypes {
		if v == mimeType {
			goto skipMIMEAdd
		}
	}
	// add
	mimeTypes[pureExt] = append(mtypes, mimeType)
skipMIMEAdd:

	if ext == "*" || (len(ext) != 0 && ext[0] == '!') {
		return
	}

	exts := mimeExtensions[justType]
	for _, v := range exts {
		if v == pureExt {
			return
		}
	}
	// add
	mimeExtensions[justType] = append(exts, pureExt)
	return
}

func mimeTypesByExtension(ext string) []string {
	// case-sensitive lookup
	if v, ok := mimeTypes[ext]; ok && len(v) != 0 {
		return v
	}
	// case-insensitive lookup
	// Optimistically assume a short ASCII extension and be
	// allocation-free in that case.
	var buf [10]byte
	lower := buf[:0]
	const utf8RuneSelf = 0x80 // from utf8 package, but not importing it.
	for i := 0; i < len(ext); i++ {
		c := ext[i]
		if c >= utf8RuneSelf {
			// Slow path.
			if v, ok := mimeTypes[strings.ToLower(ext)]; ok && len(v) != 0 {
				return v
			}
			goto notFound
		}
		if 'A' <= c && c <= 'Z' {
			lower = append(lower, c+('a'-'A'))
		} else {
			lower = append(lower, c)
		}
	}
	if v, ok := mimeTypes[string(lower)]; ok && len(v) != 0 {
		return v
	}
notFound:
	if v, ok := mimeTypes["*"]; ok && len(v) != 0 {
		return v
	}
	return nil
}

func mimeTypeByExtension(ext string) string {
	if typ := mimeTypesByExtension(ext); len(typ) != 0 {
		return typ[0]
	}
	return ""
}

// MIMETypeByExtension takes extension (without dot)
// and returns first MIME type for it. If no extension, pass empty string.
// Returns empty string on failure.
func MIMETypeByExtension(ext string) string {
	mimeLock.RLock()
	typ := mimeTypeByExtension(ext)
	mimeLock.RUnlock()
	return typ
}

func mimeIsCanonical(ext, typ string) bool {
	ext = strings.ToLower(ext)
	if typext, err := mimeExtensionsByType(typ); err == nil {
		for _, tex := range typext {
			if ext == tex {
				return true
			}
		}
	}
	return false
}

// MIMEIsCanonical tells whether ext is one of MIME type typ extensions.
// Canonical means that this extension is gettable by MIME type.
// Some extensions lead to certain MIME types which aren't official.
func MIMEIsCanonical(ext, typ string) bool {
	mimeLock.RLock()
	can := mimeIsCanonical(ext, typ)
	mimeLock.RUnlock()
	return can
}

func mimeCanonicalTypeByExtension(ext string) string {
	typ := mimeTypesByExtension(ext)
	for _, t := range typ {
		if mimeIsCanonical(ext, t) {
			return t
		}
	}
	return ""
}

// MIMECanonicalTypeByExtension returns canonical MIME type
// for given extension.
func MIMECanonicalTypeByExtension(ext string) string {
	mimeLock.RLock()
	typ := mimeCanonicalTypeByExtension(ext)
	mimeLock.RUnlock()
	return typ
}

func mimeExtensionsByType(mimeType string) ([]string, error) {
	justType, _, err := mm.ParseMediaType(mimeType)
	if err != nil {
		return nil, err
	}
	s := mimeExtensions[justType]
	return s, nil
}

// MIMEExtensionsByType takes MIME type and returns extensions (without dot)
// for it.
func MIMEExtensionsByType(mimeType string) (ext []string, err error) {
	mimeLock.RLock()
	ext, err = mimeExtensionsByType(mimeType)
	mimeLock.RUnlock()
	return
}

// LoadMIMEDatabase loads MIME database from specified path.
// Extensions may start with "." which will be ignored.
// Specify wildcard extensions with "*", empty extensions as ".",
// start non-canonical extensions with "!".
// "!" alone can be used for empty non-canonical extension.
// Types like "application/octet-stream" should use non-canonical extensions.
func LoadMIMEDatabase(dbfile string) (err error) {
	mimeLock.Lock()
	defer mimeLock.Unlock()

	mimeTypes = make(map[string][]string)
	mimeExtensions = make(map[string][]string)

	if dbfile == "" {
		return nil
	}
	f, err := os.Open(dbfile)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) <= 1 || fields[0][0] == '#' || fields[0][0] == '/' {
			continue
		}
		mimeType := fields[0]
		e := setExtensionsType(fields[1:], mimeType)
		if err == nil {
			err = e
		}
	}
	if e := scanner.Err(); e != nil {
		err = e
	}
	return
}

func setExtensionsType(exts []string, mimeType string) (err error) {
	numpref := -1
	for i, ext := range exts {
		if ext[0] == '#' || ext[0] == '/' {
			break
		}

		ext, pref, e := setExtensionType(strings.ToLower(ext), mimeType)
		exts[i] = ext

		if err == nil {
			err = e
		}
		if pref && numpref < 0 {
			numpref = i
		}
	}
	if numpref >= 0 && exts[numpref] != "" {
		for i, ext := range exts {
			if i == numpref || ext == "" {
				continue
			}
			if ext[0] == '#' || ext[0] == '/' {
				break
			}
			_, exists := mimePrefExt[ext]
			if !exists {
				mimePrefExt[ext] = exts[numpref]
			}
		}
	}
	return
}
