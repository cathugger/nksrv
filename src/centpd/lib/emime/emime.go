package emime

import (
	"bufio"
	"errors"
	"io"
	mm "mime"
	"os"
	"strings"
	"sync"
)

var errNoInit = errors.New("emime not initialized")

var (
	mimeLock       sync.RWMutex
	initialized    bool
	mimeTypes      map[string][]string // extension -> types
	mimeExtensions map[string][]string // type -> extensions
	mimePrefExt    map[string]string   // prefered extensions
)

// ext MUST be lowercase
func setExtensionTypeLocked(
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

func mimeTypesByExtensionLocked(ext string) []string {
	// case-sensitive lookup
	if v, ok := mimeTypes[ext]; ok {
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
			if v, ok := mimeTypes[strings.ToLower(ext)]; ok {
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
	if v, ok := mimeTypes[string(lower)]; ok {
		return v
	}
notFound:
	if v, ok := mimeTypes["*"]; ok {
		return v
	}
	return nil
}

func mimeTypeByExtensionLocked(ext string) string {
	if typ := mimeTypesByExtensionLocked(ext); len(typ) != 0 {
		return typ[0]
	}
	return ""
}

// MIMETypeByExtension takes extension (without dot)
// and returns first MIME type for it. If no extension, pass empty string.
// Returns empty string on failure.
func MIMETypeByExtension(ext string) string {
	mimeLock.RLock()

	if !initialized {
		mimeLock.RUnlock()
		panic(errNoInit)
	}

	typ := mimeTypeByExtensionLocked(ext)

	mimeLock.RUnlock()

	return typ
}

func mimeIsCanonicalLocked(ext, typ string) bool {
	ext = strings.ToLower(ext)
	if typext, err := mimeExtensionsByTypeLocked(typ); err == nil {
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

	if !initialized {
		mimeLock.RUnlock()
		panic(errNoInit)
	}

	can := mimeIsCanonicalLocked(ext, typ)

	mimeLock.RUnlock()

	return can
}

func mimeCanonicalTypeByExtensionLocked(ext string) string {
	typ := mimeTypesByExtensionLocked(ext)
	for _, t := range typ {
		if mimeIsCanonicalLocked(ext, t) {
			return t
		}
	}
	return ""
}

// MIMECanonicalTypeByExtension returns canonical MIME type
// for given extension.
func MIMECanonicalTypeByExtension(ext string) string {
	mimeLock.RLock()

	if !initialized {
		mimeLock.RUnlock()
		panic(errNoInit)
	}

	typ := mimeCanonicalTypeByExtensionLocked(ext)

	mimeLock.RUnlock()

	return typ
}

func mimeExtensionsByTypeLocked(mimeType string) ([]string, error) {
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

	if !initialized {
		mimeLock.RUnlock()
		panic(errNoInit)
	}

	ext, err = mimeExtensionsByTypeLocked(mimeType)

	mimeLock.RUnlock()

	return
}

// LoadMIMEDatabase loads MIME database from specified path.
// Extensions may start with "." which will be ignored.
// Specify wildcard extensions with "*", empty extensions as ".",
// start non-canonical extensions with "!", start prefered extensions with "=".
// "!" alone can be used for empty non-canonical extension.
// Prefered extension should be specified first.
// Types like "application/octet-stream" should use non-canonical extensions.
func LoadMIMEDatabase(dbfile ...string) (err error) {
	mimeLock.Lock()
	defer mimeLock.Unlock()

	// initialize maps
	initialized = true
	mimeTypes = make(map[string][]string)
	mimeExtensions = make(map[string][]string)
	mimePrefExt = make(map[string]string)

	for _, fname := range dbfile {
		err = loadMIMEDatabaseFromFileLocked(fname)
		if err != nil {
			if os.IsNotExist(err) {
				// ignore file doesn't exist error
				err = nil
				continue
			}
			return
		}
	}

	return
}

func loadMIMEDatabaseFromFileLocked(dbfile string) (err error) {
	if dbfile == "" {
		return nil
	}

	f, err := os.Open(dbfile)
	if err != nil {
		return
	}

	err = loadMIMEDatabaseFromReaderLocked(f)

	f.Close()

	return
}

func loadMIMEDatabaseFromReaderLocked(r io.Reader) (err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) <= 1 || fields[0][0] == '#' || fields[0][0] == '/' {
			continue
		}
		mimeType := fields[0]
		e := setExtensionsTypeLocked(fields[1:], mimeType)
		if err == nil {
			err = e
		}
	}
	if e := scanner.Err(); e != nil {
		err = e
	}
	return
}

func setExtensionsTypeLocked(exts []string, mimeType string) (err error) {

	prefext := ""

	for i, ext := range exts {
		if ext[0] == '#' || ext[0] == '/' {
			break
		}

		ext, pref, e := setExtensionTypeLocked(strings.ToLower(ext), mimeType)

		if err == nil {
			err = e
		}

		if pref && ext != "" {
			if prefext == "" {
				prefext = ext
			}
			// handle more than one prefered extension
			exts[i] = ""
		} else {
			exts[i] = ext
		}
	}

	if prefext != "" {
		for _, ext := range exts {
			if ext == "" {
				continue
			}
			if ext[0] == '#' || ext[0] == '/' {
				break
			}
			_, exists := mimePrefExt[ext]
			if !exists {
				mimePrefExt[ext] = prefext
			}
		}
	}

	return
}

// MIMEGetPreferedExtension gets prefered extension.
// As side effect, it also forces extension lowercase.
func MIMEPreferedExtension(ext string) string {
	ext = strings.ToLower(ext)

	mimeLock.RLock()

	if !initialized {
		mimeLock.RUnlock()
		panic(errNoInit)
	}

	pext := mimePrefExt[ext]

	mimeLock.RUnlock()

	if pext != "" {
		return pext
	} else {
		return ext
	}
}
